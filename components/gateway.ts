import * as pulumi from "@pulumi/pulumi";
import * as docker from "@pulumi/docker";
import * as command from "@pulumi/command";
import {
  TailscaleMode,
  DEFAULT_OPENCLAW_CONFIG_DIR,
  DEFAULT_OPENCLAW_WORKSPACE_DIR,
  ENVOY_STATIC_IP,
  ENVOY_CA_CERT_PATH,
  TAILSCALE_STATE_DIR,
  TAILSCALE_SOCKET_PATH,
} from "../config";
import {
  renderDockerfile,
  renderEntrypoint,
  TcpPortMapping,
  UdpPortMapping,
} from "../templates";

export interface GatewayArgs {
  /** Docker host URI, e.g. "ssh://root@<ip>" */
  dockerHost: pulumi.Input<string>;
  /** SSH connection args for remote commands */
  connection: pulumi.Input<command.types.input.remote.ConnectionArgs>;
  /** Internal network name (from EnvoyEgress) — gateway attaches here */
  internalNetworkName: pulumi.Input<string>;
  /** Unique name for this gateway instance */
  profile: string;
  /** OpenClaw version to install (npm dist-tag or semver) */
  version: string;
  /** Additional apt packages to bake into the image */
  packages: string[];
  /** Host port for the gateway */
  port: number;
  /** Bridge port (defaults 18790) */
  bridgePort?: number;
  /** Tailscale mode: "serve" (private), "funnel" (public), or "off" */
  tailscale: TailscaleMode;
  /** Bake Playwright + Chromium into the image (~300MB) */
  installBrowser?: boolean;
  /** openclaw config set key=value pairs (user overrides, cannot override security-critical keys) */
  configSet: Record<string, string>;
  /** OpenClaw subcommands run in the init container after config set (auto-prefixed with `openclaw `) */
  setupCommands?: string[];
  /** Additional env vars for the container */
  env?: Record<string, string>;
  /** Secret env vars (JSON string: {"KEY":"value",...}) for init container and main container */
  secretEnv?: pulumi.Input<string>;
  /** Auth configuration for this gateway */
  auth: { mode: "token"; token: pulumi.Input<string> };
  /** Per-rule port mappings for SSH/TCP egress (from EnvoyEgress) */
  tcpPortMappings?: TcpPortMapping[];
  /** Per-rule port mappings for UDP egress (from EnvoyEgress) */
  udpPortMappings?: UdpPortMapping[];
  /** Secret: Tailscale auth key (required when tailscale != "off") */
  tailscaleAuthKey?: pulumi.Input<string>;
}

export class Gateway extends pulumi.ComponentResource {
  /** Docker container ID */
  public readonly containerId: pulumi.Output<string>;
  /** Tailscale hostname resolved from the container (empty string if tailscale is "off") */
  public readonly tailscaleUrl: pulumi.Output<string>;

  constructor(
    name: string,
    args: GatewayArgs,
    opts?: pulumi.ComponentResourceOptions,
  ) {
    super("openclaw:app:Gateway", name, {}, opts);

    const buildDir = `/opt/openclaw-deploy/build/${args.profile}`;
    const dataDir = `/opt/openclaw-deploy/data/${args.profile}`;
    const tailscaleEnabled = args.tailscale !== "off";

    // Render templates (pure functions, runs at plan time)
    const dockerfile = renderDockerfile({
      version: args.version,
      packages: args.packages,
      installBrowser: args.installBrowser ?? false,
      bridgePort: args.bridgePort,
    });
    const entrypoint = renderEntrypoint();

    // Docker provider connected to the remote host
    const dockerProvider = new docker.Provider(
      `${name}-docker`,
      { host: args.dockerHost },
      { parent: this },
    );

    // Step 1: Upload Dockerfile + entrypoint.sh to server via remote command.
    // Uses base64 encoding to safely transfer content without heredoc issues.
    const encodedDockerfile = Buffer.from(dockerfile).toString("base64");
    const encodedEntrypoint = Buffer.from(entrypoint).toString("base64");
    const uploadBuildContext = new command.remote.Command(
      `${name}-upload-build`,
      {
        connection: args.connection,
        create: [
          `mkdir -p ${buildDir}`,
          `echo '${encodedDockerfile}' | base64 -d > ${buildDir}/Dockerfile`,
          `echo '${encodedEntrypoint}' | base64 -d > ${buildDir}/entrypoint.sh`,
          `chmod 755 ${buildDir}/entrypoint.sh`,
        ].join(" && "),
        delete: `rm -rf ${buildDir}`,
      },
      { parent: this },
    );

    // Step 2: Build Docker image on the remote host via docker build command.
    // Cannot use docker.Image because it validates the Dockerfile path locally
    // during preview, but the build context only exists on the remote host.
    const imageName = `openclaw-gateway-${args.profile}:${args.version}`;
    const buildImage = new command.remote.Command(
      `${name}-build-image`,
      {
        connection: args.connection,
        create: `docker build --platform linux/amd64 -t ${imageName} ${buildDir}`,
        delete: `docker rmi ${imageName} 2>/dev/null; true`,
      },
      {
        parent: this,
        dependsOn: [uploadBuildContext],
      },
    );

    // Step 3: Create host directories for persistent data
    const mkdirParts = [`${dataDir}/{config,workspace,config/identity}`];
    if (tailscaleEnabled) {
      mkdirParts.push(`${dataDir}/tailscale`);
    }
    const createDirs = new command.remote.Command(
      `${name}-dirs`,
      {
        connection: args.connection,
        create: `mkdir -p ${mkdirParts.join(" ")} && chown -R 1000:1000 ${dataDir}`,
        delete: `rm -rf ${dataDir}`,
      },
      { parent: this },
    );

    // Step 4: Write config to shared volume via ephemeral CLI container.
    // Runs BEFORE the gateway container starts — avoids crash-loop from missing config.
    // Uses --network none (pure file I/O) and --user node (no root-owned files).
    const containerName = `openclaw-gateway-${args.profile}`;

    // Security-critical config — user configSet cannot override these keys.
    const requiredConfig: Record<string, string> = {
      "gateway.mode": "local",
      "gateway.trustedProxies":
        '["172.16.0.0/12","10.0.0.0/8","192.168.0.0/16"]',
      "discovery.mdns.mode": "off",
    };

    // User overrides first, then required config on top (required always wins)
    const allConfig: Record<string, string> = {
      ...args.configSet,
      ...requiredConfig,
    };

    const configCmds = Object.entries(allConfig).map(
      ([key, value]) =>
        `openclaw config set ${key} '${value.replace(/'/g, "'\\''")}'`,
    );

    // Append user setup commands (prefixed with `openclaw `, can reference $SECRET_ENV_VARS)
    const setupCmds = (args.setupCommands ?? []).map(
      (cmd) => `openclaw ${cmd}`,
    );
    const allInitCmds = [...configCmds, ...setupCmds];

    // Base64-encode the init script to avoid nested shell quoting issues.
    // The script runs inside the container where env vars ($SECRET_ENV_VARS)
    // are available from the --env-file.
    const initScript = allInitCmds.join("\n");
    const encodedInitScript = Buffer.from(initScript).toString("base64");

    // Step 4a: Write secret env file to host (separate command so secrets
    // don't appear in the init container command string on error).
    const envFile = `${dataDir}/.init-env`;
    const writeSecretEnv = new command.remote.Command(
      `${name}-write-secret-env`,
      {
        connection: args.connection,
        create: pulumi.output(args.secretEnv ?? "{}").apply((secretJson) => {
          const secrets = JSON.parse(secretJson) as Record<string, string>;
          const entries = Object.entries(secrets);
          if (entries.length === 0)
            return `touch ${envFile} && chmod 600 ${envFile}`;
          return (
            entries
              .map(
                ([k, v]) => `printf '%s\\n' '${k}=${v.replace(/'/g, "'\\''")}'`,
              )
              .join(" && ") + ` > ${envFile} && chmod 600 ${envFile}`
          );
        }),
        delete: `rm -f ${envFile}`,
        logging: "none",
      },
      {
        parent: this,
        dependsOn: [buildImage, createDirs],
        additionalSecretOutputs: ["stdout", "stderr"],
      },
    );

    // Step 4b: Run init container with base64-decoded script piped to sh.
    // The --env-file provides secret env vars; the command string has no secrets.
    const writeConfig = new command.remote.Command(
      `${name}-write-config`,
      {
        connection: args.connection,
        create: [
          `echo '${encodedInitScript}' | base64 -d > ${dataDir}/.init.sh`,
          `&&`,
          `docker run --rm --network none --user node`,
          `--entrypoint /bin/sh`,
          `--env-file ${envFile}`,
          `-v ${dataDir}/config:${DEFAULT_OPENCLAW_CONFIG_DIR}`,
          `-v ${dataDir}/workspace:${DEFAULT_OPENCLAW_WORKSPACE_DIR}`,
          `-v ${dataDir}/.init.sh:/tmp/init.sh:ro`,
          `${imageName} /tmp/init.sh`,
          `&& rm -f ${dataDir}/.init.sh`,
        ].join(" "),
        delete: `rm -f ${dataDir}/.init.sh`,
      },
      {
        parent: this,
        dependsOn: [writeSecretEnv],
      },
    );

    // Step 5: Create the gateway container

    // Build env vars list
    const envs: pulumi.Input<string>[] = [
      `HOME=/home/node`,
      `TERM=xterm-256color`,
      // Always set so the gateway trusts MITM-issued certs (harmless when no inspect rules exist)
      `NODE_EXTRA_CA_CERTS=${ENVOY_CA_CERT_PATH}`,
    ];

    // Auth token via env var (takes precedence over config file in local mode)
    envs.push(pulumi.interpolate`OPENCLAW_GATEWAY_TOKEN=${args.auth.token}`);

    if (args.tcpPortMappings && args.tcpPortMappings.length > 0) {
      envs.push(
        `OPENCLAW_TCP_MAPPINGS=${args.tcpPortMappings.map((m) => `${m.dst}|${m.dstPort}|${m.envoyPort}`).join(";")}`,
      );
    }

    if (args.udpPortMappings && args.udpPortMappings.length > 0) {
      envs.push(
        `OPENCLAW_UDP_MAPPINGS=${args.udpPortMappings.map((m) => `${m.dst}|${m.dstPort}|${m.envoyPort}`).join(";")}`,
      );
    }

    // Tailscale env vars (secret authkey + socket path for CLI)
    if (tailscaleEnabled) {
      envs.push(`TS_SOCKET=${TAILSCALE_SOCKET_PATH}`);
      if (args.tailscaleAuthKey) {
        envs.push(
          pulumi.interpolate`TAILSCALE_AUTHKEY=${args.tailscaleAuthKey}`,
        );
      }
    }

    for (const [k, v] of Object.entries(args.env ?? {})) {
      envs.push(`${k}=${v}`);
    }

    // Merge secret env vars into the container's envs.
    // secretEnv is a Pulumi secret, so we resolve both base envs and parsed
    // secrets into a single Output<string[]> for the container's envs field.
    const secretEnvParsed = pulumi
      .output(args.secretEnv ?? "{}")
      .apply((s) => JSON.parse(s) as Record<string, string>);

    const computedEnvs = pulumi
      .all([pulumi.all(envs), secretEnvParsed])
      .apply(([baseEnvs, secrets]) => [
        ...baseEnvs,
        ...Object.entries(secrets).map(([k, v]) => `${k}=${v}`),
      ]);

    // Build volumes list
    const volumes: docker.types.input.ContainerVolume[] = [
      {
        hostPath: `${dataDir}/config`,
        containerPath: DEFAULT_OPENCLAW_CONFIG_DIR,
      },
      {
        hostPath: `${dataDir}/workspace`,
        containerPath: DEFAULT_OPENCLAW_WORKSPACE_DIR,
      },
      {
        hostPath: ENVOY_CA_CERT_PATH,
        containerPath: ENVOY_CA_CERT_PATH,
        readOnly: true,
      },
    ];

    if (tailscaleEnabled) {
      volumes.push({
        hostPath: `${dataDir}/tailscale`,
        containerPath: TAILSCALE_STATE_DIR,
      });
    }

    // Container command depends on Tailscale mode.
    // Config is already written to the shared volume by writeConfig.
    const containerCommand = tailscaleEnabled
      ? [
          "openclaw",
          "gateway",
          "--tailscale",
          args.tailscale,
          "--port",
          `${args.port}`,
        ]
      : ["openclaw", "gateway", "--bind", "lan", "--port", `${args.port}`];

    const container = new docker.Container(
      `${name}-container`,
      {
        name: containerName,
        image: imageName,
        restart: "unless-stopped",
        init: true,
        capabilities: { adds: ["NET_ADMIN"] },
        dns: [ENVOY_STATIC_IP],
        envs: computedEnvs,
        command: containerCommand,
        volumes,
        networksAdvanced: [{ name: args.internalNetworkName }],
      },
      {
        parent: this,
        provider: dockerProvider,
        dependsOn: [writeConfig],
        additionalSecretOutputs: ["envs"],
      },
    );

    // Step 6: Query Tailscale hostname from inside the container (if not "off")
    if (tailscaleEnabled) {
      const tailscaleHostname = new command.remote.Command(
        `${name}-tailscale-url`,
        {
          connection: args.connection,
          create: [
            // Wait for Tailscale to authenticate inside the container (up to 120s)
            `for i in $(seq 1 60); do docker exec ${containerName} tailscale --socket=/var/run/tailscale/tailscaled.sock status --json 2>/dev/null | jq -e '.BackendState == "Running"' >/dev/null 2>&1 && break; sleep 2; done`,
            // Extract DNSName via jq (installed on host by bootstrap)
            `docker exec ${containerName} tailscale --socket=/var/run/tailscale/tailscaled.sock status --json | jq -r '.Self.DNSName' | sed 's/\\.$//'`,
          ].join(" && "),
        },
        {
          parent: this,
          dependsOn: [container],
        },
      );

      this.tailscaleUrl = tailscaleHostname.stdout.apply(
        (hostname) => `https://${hostname.trim()}`,
      );
    } else {
      this.tailscaleUrl = pulumi.output("");
    }

    // Outputs
    this.containerId = container.id;

    this.registerOutputs({
      containerId: this.containerId,
      tailscaleUrl: this.tailscaleUrl,
    });
  }
}
