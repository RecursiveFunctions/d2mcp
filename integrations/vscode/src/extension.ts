import { execFile } from "node:child_process";
import { promisify } from "node:util";
import * as vscode from "vscode";
import { dockerConfiguration, portableConfiguration } from "./configuration";
import { version } from "./release";

const execFileAsync = promisify(execFile);
const providerID = "d2mcp.docker";

export function activate(context: vscode.ExtensionContext): void {
  const definitionsChanged = new vscode.EventEmitter<void>();

  context.subscriptions.push(
    definitionsChanged,
    vscode.workspace.onDidChangeWorkspaceFolders(() => definitionsChanged.fire()),
    vscode.lm.registerMcpServerDefinitionProvider(providerID, {
      onDidChangeMcpServerDefinitions: definitionsChanged.event,
      provideMcpServerDefinitions: () => definitions(),
      resolveMcpServerDefinition: async (server) => {
        await requireDocker();
        return server;
      },
    }),
    vscode.commands.registerCommand("d2mcp.showConfiguration", showConfiguration),
    vscode.commands.registerCommand("d2mcp.openDocumentation", () =>
      vscode.env.openExternal(vscode.Uri.parse("https://github.com/RecursiveFunctions/d2mcp")),
    ),
  );
}

function definitions(): vscode.McpServerDefinition[] {
  const folders = vscode.workspace.workspaceFolders ?? [];
  return folders.map((folder) => {
    const configuration = dockerConfiguration(folder.uri.fsPath);
    const label = folders.length === 1 ? "D2 MCP Server" : `D2 MCP Server (${folder.name})`;
    return new vscode.McpStdioServerDefinition(
      label,
      configuration.command,
      configuration.args,
      undefined,
      version,
    );
  });
}

async function requireDocker(): Promise<void> {
  try {
    await execFileAsync("docker", ["version", "--format", "{{.Client.Version}}"], { timeout: 10_000 });
  } catch (error) {
    const detail = error instanceof Error ? error.message : String(error);
    throw new Error(`Docker is required to run D2 MCP Server. ${detail}`);
  }
}

async function showConfiguration(): Promise<void> {
  const folders = vscode.workspace.workspaceFolders ?? [];
  if (folders.length === 0) {
    await vscode.window.showErrorMessage("Open a workspace folder before configuring D2 MCP Server.");
    return;
  }

  let folder = folders[0];
  if (folders.length > 1) {
    folder = await vscode.window.showWorkspaceFolderPick({ placeHolder: "Select the workspace for D2 MCP Server" });
  }
  if (!folder) {
    return;
  }

  const document = await vscode.workspace.openTextDocument({
    language: "jsonc",
    content: portableConfiguration(folder.uri.fsPath),
  });
  await vscode.window.showTextDocument(document);
}