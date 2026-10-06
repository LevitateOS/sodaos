export interface ConnectionInput {
  hasPage: () => boolean;
  isBlocked: () => boolean;
  readConnection: () => {command: string; fingerprint: string} | undefined;
  isAccessSelected: () => boolean;
  isConcealed: () => boolean;
  isDisposed: () => boolean;
  setOutcome: (outcome: string) => void;
}

function copyBlocked(input: ConnectionInput) {
  return (
    !input.hasPage() || input.isBlocked() || !input.readConnection() || !input.isAccessSelected() || input.isConcealed()
  );
}

function noteCopy(input: ConnectionInput, ok: boolean) {
  if (input.isDisposed()) return;
  input.setOutcome(ok ? 'SSH connection copied.' : 'Copy failed. Select and copy the displayed SSH command.');
}

export async function copyConnection(input: ConnectionInput) {
  if (copyBlocked(input)) return;
  try {
    await navigator.clipboard.writeText(input.readConnection()!.command);
    noteCopy(input, true);
  } catch {
    noteCopy(input, false);
  }
}
