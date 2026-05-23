export interface DialogAlertOptions {
  title?: string;
  message: string;
  okText?: string;
  variant?: 'default' | 'success' | 'danger' | 'warning' | 'info';
  size?: 'sm' | 'md' | 'lg';
}

export interface DialogConfirmOptions {
  title?: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: 'default' | 'danger' | 'warning' | 'info';
  size?: 'sm' | 'md' | 'lg';
}

export interface DialogPromptOptions {
  title?: string;
  message?: string;
  placeholder?: string;
  inputValue?: string;
  okText?: string;
  cancelText?: string;
  inputType?: 'text' | 'number' | 'password';
  required?: boolean;
  size?: 'sm' | 'md' | 'lg';
}

function createOverlay(): HTMLDivElement {
  const overlay = document.createElement('div');
  overlay.className = 'bc-overlay';
  overlay.style.cssText = 'position:fixed;inset:0;background:rgba(0,0,0,0.4);display:flex;align-items:center;justify-content:center;z-index:var(--bc-z-modal,300)';
  return overlay;
}

function createDialog(size: string = 'sm'): HTMLDivElement {
  const dialog = document.createElement('div');
  dialog.className = `bc-dialog bc-dialog-${size}`;
  dialog.style.cssText = 'background:var(--bc-bg);border-radius:var(--bc-radius-xl);box-shadow:var(--bc-shadow-xl);min-width:24rem;max-width:90vw;max-height:90vh;overflow:auto;padding:var(--bc-spacing-md)';
  return dialog;
}

function createHeader(title: string, onClose: () => void): HTMLDivElement {
  const header = document.createElement('div');
  header.style.cssText = 'display:flex;justify-content:space-between;align-items:center;margin-bottom:var(--bc-spacing-md)';

  const h3 = document.createElement('h3');
  h3.style.cssText = 'margin:0;font-size:var(--bc-font-size-lg)';
  h3.textContent = title;
  header.appendChild(h3);

  const closeBtn = document.createElement('button');
  closeBtn.type = 'button';
  closeBtn.innerHTML = '&times;';
  closeBtn.style.cssText = 'background:none;border:none;font-size:1.5rem;cursor:pointer;color:var(--bc-text-secondary);padding:0;line-height:1';
  closeBtn.addEventListener('click', onClose);
  header.appendChild(closeBtn);

  return header;
}

function createFooter(buttons: Array<{text: string; className: string; onClick: () => void}>): HTMLDivElement {
  const footer = document.createElement('div');
  footer.style.cssText = 'display:flex;justify-content:flex-end;gap:var(--bc-spacing-sm);margin-top:var(--bc-spacing-md);padding-top:var(--bc-spacing-md);border-top:1px solid var(--bc-border-color)';

  buttons.forEach(({text, className, onClick}) => {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.textContent = text;
    btn.className = className;
    btn.style.cssText = 'padding:0.5rem 1rem;border:1px solid var(--bc-border-color);border-radius:var(--bc-radius-md);background:var(--bc-bg);color:var(--bc-text);font-size:var(--bc-font-size-sm);cursor:pointer;transition:all var(--bc-transition)';
    if (className.includes('primary')) {
      btn.style.cssText += ';background:var(--bc-primary);color:white;border-color:var(--bc-primary)';
    } else if (className.includes('danger')) {
      btn.style.cssText += ';background:var(--bc-danger);color:white;border-color:var(--bc-danger)';
    }
    btn.addEventListener('click', onClick);
    footer.appendChild(btn);
  });

  return footer;
}

const BcDialog = {
  alert(options: DialogAlertOptions): Promise<void> {
    return new Promise((resolve) => {
      const overlay = createOverlay();
      const dialog = createDialog(options.size);
      const close = () => { overlay.remove(); resolve(); };

      const header = createHeader(options.title || '', close);
      dialog.appendChild(header);

      if (options.message) {
        const msg = document.createElement('p');
        msg.style.cssText = 'margin:0;color:var(--bc-text-secondary);font-size:var(--bc-font-size-sm)';
        msg.textContent = options.message;
        dialog.appendChild(msg);
      }

      const footer = createFooter([
        { text: options.okText || 'OK', className: 'bc-btn-primary', onClick: close }
      ]);
      dialog.appendChild(footer);

      overlay.appendChild(dialog);
      overlay.addEventListener('click', (e) => { if (e.target === overlay) close(); });
      document.body.appendChild(overlay);

      const okBtn = footer.querySelector('button');
      if (okBtn) okBtn.focus();
    });
  },

  confirm(options: DialogConfirmOptions): Promise<boolean> {
    return new Promise((resolve) => {
      const overlay = createOverlay();
      const dialog = createDialog(options.size);
      const close = (result: boolean) => { overlay.remove(); resolve(result); };

      const header = createHeader(options.title || '', () => close(false));
      dialog.appendChild(header);

      if (options.message) {
        const msg = document.createElement('p');
        msg.style.cssText = 'margin:0;color:var(--bc-text-secondary);font-size:var(--bc-font-size-sm)';
        msg.textContent = options.message;
        dialog.appendChild(msg);
      }

      const footer = createFooter([
        { text: options.cancelText || 'Cancel', className: 'bc-btn', onClick: () => close(false) },
        { text: options.confirmText || 'Confirm', className: options.variant === 'danger' ? 'bc-btn-danger' : 'bc-btn-primary', onClick: () => close(true) }
      ]);
      dialog.appendChild(footer);

      overlay.appendChild(dialog);
      overlay.addEventListener('click', (e) => { if (e.target === overlay) close(false); });
      document.body.appendChild(overlay);

      const confirmBtn = footer.querySelectorAll('button')[1];
      if (confirmBtn) confirmBtn.focus();
    });
  },

  prompt(options: DialogPromptOptions): Promise<string | null> {
    return new Promise((resolve) => {
      const overlay = createOverlay();
      const dialog = createDialog(options.size);
      let value = options.inputValue || '';
      const close = (result: string | null) => { overlay.remove(); resolve(result); };

      const header = createHeader(options.title || '', () => close(null));
      dialog.appendChild(header);

      if (options.message) {
        const msg = document.createElement('p');
        msg.style.cssText = 'margin:0;color:var(--bc-text-secondary);font-size:var(--bc-font-size-sm)';
        msg.textContent = options.message;
        dialog.appendChild(msg);
      }

      const input = document.createElement('input');
      input.type = options.inputType || 'text';
      input.placeholder = options.placeholder || '';
      input.value = value;
      input.style.cssText = 'width:100%;height:var(--bc-input-height,2.5rem);padding:0.5rem 0.75rem;border:1px solid var(--bc-border-color);border-radius:var(--bc-input-radius,0.375rem);background:var(--bc-input-bg,#fff);font-family:inherit;font-size:var(--bc-font-size-base,1rem);color:var(--bc-text);outline:none;box-sizing:border-box;margin-top:var(--bc-spacing-sm)';
      input.addEventListener('input', () => { value = input.value; });
      input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && (!options.required || value.trim())) close(value);
        if (e.key === 'Escape') close(null);
      });
      dialog.appendChild(input);

      const footer = createFooter([
        { text: options.cancelText || 'Cancel', className: 'bc-btn', onClick: () => close(null) },
        { text: options.okText || 'OK', className: 'bc-btn-primary', onClick: () => {
          if (options.required && !value.trim()) return;
          close(value);
        }}
      ]);
      dialog.appendChild(footer);

      overlay.appendChild(dialog);
      overlay.addEventListener('click', (e) => { if (e.target === overlay) close(null); });
      document.body.appendChild(overlay);

      input.focus();
    });
  },
};

if (typeof window !== 'undefined') {
  (window as any).BcDialog = BcDialog;
}

export { BcDialog };
export default BcDialog;
