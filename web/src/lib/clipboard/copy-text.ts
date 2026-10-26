/**
 * Copies text and says whether it worked. Browsers give the Clipboard API
 * only to secure pages (HTTPS or localhost), and Karvan is often opened
 * over plain HTTP by its IP, so there it copies the old way: from a hidden
 * field, selected, with the copy command. A false answer means the browser
 * copied nothing, and the caller should select the text for the reader.
 */
export async function copyText(text: string): Promise<boolean> {
  if (window.isSecureContext && navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Refused (no permission, the page not focused): try the old way.
    }
  }
  return copyThroughField(text);
}

function copyThroughField(text: string): boolean {
  const before = document.activeElement;
  // Inside a dialog the rest of the page is inert, so the field goes next to the button.
  const host = before?.closest('dialog, [role="dialog"]') ?? document.body;
  const field = document.createElement('textarea');
  field.value = text;
  field.readOnly = true;
  field.setAttribute('aria-hidden', 'true');
  field.style.position = 'fixed';
  field.style.insetBlockStart = '0';
  field.style.opacity = '0';
  host.append(field);
  field.focus();
  field.select();
  field.setSelectionRange(0, text.length);
  let copied: boolean;
  try {
    copied = document.execCommand('copy');
  } catch {
    copied = false;
  }
  field.remove();
  if (before instanceof HTMLElement) before.focus();
  return copied;
}
