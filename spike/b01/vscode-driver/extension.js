// Drives Copilot Chat for vscode-hooks-run.sh. Every second it looks at the
// workspace's .queue folder: <name>.txt is sent as a prompt in a new agent
// chat and deleted; a file named close closes the window.
const fs = require("fs");
const path = require("path");
const vscode = require("vscode");

async function send(queue, name) {
  const file = path.join(queue, name);
  const prompt = fs.readFileSync(file, "utf8").trim();
  fs.unlinkSync(file);
  await vscode.commands.executeCommand("workbench.action.chat.newChat");
  await new Promise((r) => setTimeout(r, 1000));
  await vscode.commands.executeCommand("workbench.action.chat.open", { query: prompt, mode: "agent" });
  fs.appendFileSync(path.join(queue, "sent.log"), `${Date.now()} ${name}\n`);
}

function activate() {
  const folder = vscode.workspace.workspaceFolders?.[0];
  if (!folder) return;
  const queue = path.join(folder.uri.fsPath, ".queue");
  fs.mkdirSync(queue, { recursive: true });
  let busy = false;
  setInterval(async () => {
    if (busy) return;
    busy = true;
    try {
      const names = fs.readdirSync(queue);
      if (names.includes("close")) {
        fs.unlinkSync(path.join(queue, "close"));
        await vscode.commands.executeCommand("workbench.action.closeWindow");
        return;
      }
      const next = names.filter((n) => n.endsWith(".txt")).sort()[0];
      if (next) await send(queue, next);
    } catch (e) {
      fs.appendFileSync(path.join(queue, "sent.log"), `error ${e}\n`);
    } finally {
      busy = false;
    }
  }, 1000);
}

module.exports = { activate };
