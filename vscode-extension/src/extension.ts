import * as fs from 'fs';
import * as path from 'path';
import * as vscode from 'vscode';
import { LanguageClient, LanguageClientOptions, ServerOptions, Trace } from 'vscode-languageclient/node';

let client: LanguageClient | undefined;

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  const serverPath = resolveServerPath(context);
  if (!serverPath) {
    void vscode.window.showWarningMessage('GoScript language server was not found. Set goscript.serverPath to your gs executable.');
    return;
  }

  const serverOptions: ServerOptions = {
    command: serverPath,
    args: ['lsp'],
    options: {
      cwd: firstWorkspaceFolder() ?? context.extensionPath
    }
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: 'file', language: 'goscript' }],
    synchronize: {
      configurationSection: 'goscript',
      fileEvents: vscode.workspace.createFileSystemWatcher('**/*.gs')
    }
  };

  client = new LanguageClient('goscriptLanguageServer', 'GoScript Language Server', serverOptions, clientOptions);
  client.setTrace(traceFromConfig());
  context.subscriptions.push(client);
  await client.start();
}

export async function deactivate(): Promise<void> {
  if (client) {
    await client.stop();
    client = undefined;
  }
}

function resolveServerPath(context: vscode.ExtensionContext): string | undefined {
  const configured = vscode.workspace.getConfiguration('goscript').get<string>('serverPath')?.trim();
  if (configured && fileExists(configured)) {
    return configured;
  }

  const candidates = new Set<string>();
  const workspace = firstWorkspaceFolder();
  if (workspace) {
    candidates.add(path.join(workspace, executableName('gs')));
    candidates.add(path.join(workspace, 'cmd', 'gs', executableName('gs')));
  }
  candidates.add(path.join(context.extensionPath, '..', executableName('gs')));
  candidates.add(path.join(context.extensionPath, '..', '..', executableName('gs')));
  candidates.add(executableName('gs'));

  for (const candidate of candidates) {
    if (candidate === executableName('gs') || fileExists(candidate)) {
      return candidate;
    }
  }
  return undefined;
}

function firstWorkspaceFolder(): string | undefined {
  return vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
}

function executableName(name: string): string {
  return process.platform === 'win32' ? `${name}.exe` : name;
}

function fileExists(file: string): boolean {
  try {
    return fs.statSync(file).isFile();
  } catch {
    return false;
  }
}

function traceFromConfig(): Trace {
  const value = vscode.workspace.getConfiguration('goscript').get<string>('trace.server');
  switch (value) {
    case 'messages':
      return Trace.Messages;
    case 'verbose':
      return Trace.Verbose;
    default:
      return Trace.Off;
  }
}
