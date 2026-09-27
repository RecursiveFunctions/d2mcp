const assert = require("node:assert/strict");
const test = require("node:test");
const { build } = require("esbuild");
const path = require("node:path");
const os = require("node:os");
const fs = require("node:fs/promises");

async function loadConfiguration() {
  const directory = await fs.mkdtemp(path.join(os.tmpdir(), "d2mcp-extension-test-"));
  const output = path.join(directory, "configuration.cjs");
  await build({
    entryPoints: [path.join(__dirname, "..", "src", "configuration.ts")],
    bundle: true,
    platform: "node",
    format: "cjs",
    outfile: output,
  });
  return require(output);
}

test("builds an immutable confined Docker command", async () => {
  const configuration = await loadConfiguration();
  const result = configuration.dockerConfiguration("C:\\work area\\diagram");
  assert.equal(result.command, "docker");
  assert.deepEqual(result.args.slice(0, 5), ["run", "--rm", "-i", "--mount", "type=bind,src=C:\\work area\\diagram,dst=/workspace"]);
  assert.equal(result.args.at(-1), configuration.image);
  assert.match(configuration.image, new RegExp(`:${configuration.version}$`));
  assert.doesNotMatch(configuration.image, /:latest$/);
});

test("rejects an empty workspace", async () => {
  const configuration = await loadConfiguration();
  assert.throws(() => configuration.dockerConfiguration("  "), /workspace folder is required/);
});

test("renders VS Code MCP configuration without losing spaces", async () => {
  const configuration = await loadConfiguration();
  const result = JSON.parse(configuration.portableConfiguration("/home/ray/project space"));
  assert.equal(result.servers.d2mcp.type, "stdio");
  assert.equal(result.servers.d2mcp.args[4], "type=bind,src=/home/ray/project space,dst=/workspace");
});