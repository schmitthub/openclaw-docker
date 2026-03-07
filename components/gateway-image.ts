import * as pulumi from "@pulumi/pulumi";
import * as command from "@pulumi/command";
import * as docker_build from "@pulumi/docker-build";
import * as fs from "fs";
import * as os from "os";
import * as path from "path";
import {
  renderDockerfile,
  renderEntrypoint,
  renderFirewallBypass,
} from "../templates";
import type { ImageStep } from "../config/types";

export interface GatewayImageArgs {
  /** SSH connection args for remote commands */
  connection: pulumi.Input<command.types.input.remote.ConnectionArgs>;
  /** Docker host URI for the remote build daemon, e.g. "ssh://root@<ip>" */
  dockerHost: pulumi.Input<string>;
  /** Unique name for this gateway instance */
  profile: string;
  /** OpenClaw version to install (npm dist-tag or semver) */
  version: string;
  /** Bake Playwright + Chromium into the image (~300MB) */
  installBrowser?: boolean;
  /** Custom Dockerfile RUN instructions (after openclaw install, before entrypoint COPY) */
  imageSteps?: ImageStep[];
  /** Push to Docker Hub instead of building on VPS. Requires DOCKERHUB_* env vars. */
  dockerhubPush?: boolean;
}

export class GatewayImage extends pulumi.ComponentResource {
  /** The image tag, e.g. "openclaw-gateway-dev:latest" or "registry/openclaw-gateway-dev:latest" */
  public readonly imageName: pulumi.Output<string>;

  constructor(
    name: string,
    args: GatewayImageArgs,
    opts?: pulumi.ComponentResourceOptions,
  ) {
    super("openclaw:build:GatewayImage", name, {}, opts);

    // Render templates (pure functions, runs at plan time)
    const dockerfile = renderDockerfile({
      version: args.version,
      installBrowser: args.installBrowser ?? false,
      imageSteps: args.imageSteps,
    });
    const entrypoint = renderEntrypoint();
    const bypassScript = renderFirewallBypass();

    // Write build context files to a stable temp directory.
    // Using a stable path (not mkdtempSync) avoids accumulating stale dirs across runs.
    // Only write if content changed — preserves mtime so BuildKit context hash is stable.
    const tempDir = path.join(os.tmpdir(), `openclaw-build-${args.profile}`);
    fs.mkdirSync(tempDir, { recursive: true });
    writeIfChanged(path.join(tempDir, "Dockerfile"), dockerfile, 0o644);
    writeIfChanged(path.join(tempDir, "entrypoint.sh"), entrypoint, 0o755);
    writeIfChanged(path.join(tempDir, "firewall-bypass"), bypassScript, 0o700);

    if (args.dockerhubPush) {
      this.imageName = this.buildAndPush(name, args, tempDir);
    } else {
      this.imageName = this.buildOnHost(name, args, tempDir);
    }

    this.registerOutputs({
      imageName: this.imageName,
    });
  }

  /** Build locally and push to Docker Hub. VPS pulls the image. */
  private buildAndPush(
    name: string,
    args: GatewayImageArgs,
    tempDir: string,
  ): pulumi.Output<string> {
    const registry = process.env.DOCKERHUB_REGISTRY;
    const username = process.env.DOCKERHUB_USERNAME;
    const token = process.env.DOCKERHUB_TOKEN;

    if (!registry || !username || !token) {
      throw new Error(
        "dockerhubPush requires DOCKERHUB_REGISTRY, DOCKERHUB_USERNAME, and DOCKERHUB_TOKEN env vars",
      );
    }

    const remoteTag = `${registry}/openclaw-gateway-${args.profile}:${args.version}`;

    // Build locally and push to Docker Hub
    const image = new docker_build.Image(
      `${name}-image`,
      {
        tags: [remoteTag],
        dockerfile: { location: path.join(tempDir, "Dockerfile") },
        context: { location: tempDir },
        push: true,
        load: false,
        buildOnPreview: false,
        registries: [
          {
            address: registry,
            username,
            password: token,
          },
        ],
      },
      { parent: this },
    );

    // Ensure VPS can pull the image (docker login on remote host).
    // Token is passed via environment to avoid shell interpolation.
    const dockerLogin = new command.remote.Command(
      `${name}-docker-login`,
      {
        connection: args.connection,
        environment: {
          DOCKERHUB_TOKEN: token,
          DOCKERHUB_REGISTRY: registry,
          DOCKERHUB_USERNAME: username,
        },
        create: `echo "$DOCKERHUB_TOKEN" | docker login "$DOCKERHUB_REGISTRY" -u "$DOCKERHUB_USERNAME" --password-stdin`,
        logging: "none",
      },
      { parent: this, additionalSecretOutputs: ["stdout", "stderr"] },
    );

    // Pull the image on the VPS
    new command.remote.Command(
      `${name}-pull`,
      {
        connection: args.connection,
        create: `docker pull ${remoteTag}`,
        triggers: [image.ref],
      },
      { parent: this, dependsOn: [image, dockerLogin] },
    );

    // Prune dangling images on VPS after pull
    new command.remote.Command(
      `${name}-prune`,
      {
        connection: args.connection,
        create: "docker image prune -f 2>&1 || true",
        triggers: [image.ref],
      },
      { parent: this, dependsOn: [image] },
    );

    return image.tags.apply((tags) => {
      if (!tags || tags.length === 0) {
        throw new Error(`No tags found for image ${name}`);
      }
      return tags[0];
    });
  }

  /**
   * Build on the VPS via DOCKER_HOST=ssh://.
   *
   * WARNING: The @pulumi/docker-build provider creates an unmanaged BuildKit
   * container on the remote host whose build cache cannot be pruned via the
   * Docker CLI (pulumi/pulumi-docker-build#65). Build cache will accumulate
   * over time. To reclaim disk space, SSH into the VPS and run:
   *
   *   docker ps --filter name=buildx_buildkit -q \
   *     | xargs -r -I{} docker exec {} buildctl prune --keep-storage 2048
   *
   * Consider setting `dockerhubPush: true` in stack config to build locally
   * and push to Docker Hub instead, which avoids this issue entirely.
   */
  private buildOnHost(
    name: string,
    args: GatewayImageArgs,
    tempDir: string,
  ): pulumi.Output<string> {
    const tag = `openclaw-gateway-${args.profile}:${args.version}`;

    pulumi.log.warn(
      [
        `[${args.profile}] Building on VPS via SSH. The @pulumi/docker-build provider creates`,
        "an unmanaged BuildKit container whose build cache accumulates on disk",
        "(pulumi/pulumi-docker-build#65). To reclaim space, SSH into the VPS and run:",
        "",
        "  docker ps --filter name=buildx_buildkit -q \\",
        "    | xargs -r -I{} docker exec {} buildctl prune --keep-storage 2048",
        "",
        "To avoid this, set `dockerhubPush: true` in stack config to build locally",
        "and push to Docker Hub instead.",
      ].join("\n"),
      this,
    );

    const buildProvider = new docker_build.Provider(
      `${name}-build-provider`,
      { host: args.dockerHost },
      { parent: this },
    );

    const image = new docker_build.Image(
      `${name}-image`,
      {
        tags: [tag],
        dockerfile: { location: path.join(tempDir, "Dockerfile") },
        context: { location: tempDir },
        load: true,
        push: false,
        buildOnPreview: false,
      },
      { parent: this, provider: buildProvider },
    );

    // Prune dangling images after build (previous untagged builds)
    new command.remote.Command(
      `${name}-prune`,
      {
        connection: args.connection,
        create: "docker image prune -f 2>&1 || true",
        triggers: [image.ref],
      },
      { parent: this, dependsOn: [image] },
    );

    return image.tags.apply((tags) => {
      if (!tags || tags.length === 0) {
        throw new Error(`No tags found for image ${name}`);
      }
      return tags[0];
    });
  }
}

/** Write file only if content differs from what's on disk — preserves mtime for stable BuildKit context hashing. */
function writeIfChanged(filePath: string, content: string, mode: number) {
  try {
    const existing = fs.readFileSync(filePath, "utf-8");
    if (existing === content) return;
  } catch (err: unknown) {
    if (
      !(
        err instanceof Error &&
        "code" in err &&
        (err as NodeJS.ErrnoException).code === "ENOENT"
      )
    ) {
      throw err;
    }
  }
  fs.writeFileSync(filePath, content, { mode });
}
