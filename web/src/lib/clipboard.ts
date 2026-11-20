// copyText tries the async clipboard API first and falls back to a hidden
// textarea, which still works in older browsers and some embedded webviews.
export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    // fall through to execCommand
  }
  const area = document.createElement('textarea');
  area.value = text;
  area.setAttribute('readonly', '');
  area.style.position = 'fixed';
  area.style.top = '-1000px';
  area.style.opacity = '0';
  document.body.appendChild(area);
  try {
    area.select();
    const ok = document.execCommand('copy');
    return ok;
  } catch {
    return false;
  } finally {
    area.remove();
  }
}
