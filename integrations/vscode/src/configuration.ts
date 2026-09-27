export { image, version } from "./release";
import { image } from "./release";

export interface DockerConfiguration {
  command: string;
  args: string[];
}

export function dockerConfiguration(workspacePath: string): DockerConfiguration {
  if (workspacePath.trim() === "") {
    throw new Error("A workspace folder is required to run d2mcp.");
  }

  return {
    command: "docker",
    args: [
      "run",
      "--rm",
      "-i",
      "--mount",
      `type=bind,src=${workspacePath},dst=/workspace`,
      image,
    ],
  };
}

export function portableConfiguration(workspacePath: string): string {
  const configuration = dockerConfiguration(workspacePath);
  return JSON.stringify({
    servers: {
      d2mcp: {
        type: "stdio",
        command: configuration.command,
        args: configuration.args,
      },
    },
  }, null, 2);
}