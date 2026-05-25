import { Component, Prop, Event, EventEmitter, Method, h } from '@stencil/core';

@Component({
  tag: 'bc-dialog-alert',
  styleUrl: 'bc-dialog-alert.css',
  shadow: false,
})
export class BcDialogAlert {
  @Prop({ mutable: true }) open: boolean = false;
  @Prop() dialogTitle: string = '';
  @Prop() message: string = '';
  @Prop() okText: string = 'OK';
  @Prop() variant: 'default' | 'success' | 'danger' | 'warning' | 'info' = 'default';
  @Prop() size: 'sm' | 'md' | 'lg' = 'sm';

  @Event() lcDialogConfirm!: EventEmitter<void>;
  @Event() lcDialogClose!: EventEmitter<{type: string}>;

  @Method() async openDialog(): Promise<void> { this.open = true; }
  @Method() async closeDialog(): Promise<void> { this._close(); }

  private _close() {
    this.open = false;
    this.lcDialogConfirm.emit();
    this.lcDialogClose.emit({type: 'alert'});
  }

  private _icon(): string {
    switch (this.variant) {
      case 'success': return '✓';
      case 'danger': return '✕';
      case 'warning': return '⚠';
      case 'info': return 'ℹ';
      default: return '💬';
    }
  }

  render() {
    if (!this.open) return null;
    const btnClass = this.variant === 'success' ? 'bc-btn-success' : this.variant === 'danger' ? 'bc-btn-danger' : this.variant === 'warning' ? 'bc-btn-warning' : this.variant === 'info' ? 'bc-btn-info' : 'bc-btn-primary';
    return (
      <div class="bc-overlay" onClick={() => this._close()}>
        <div class={`bc-dialog bc-dialog-${this.size}`} onClick={(e) => e.stopPropagation()} role="alertdialog" aria-modal="true" aria-label={this.dialogTitle}>
          <div class="bc-dialog-header">
            <h3><span class={`bc-dialog-icon bc-dialog-icon-${this.variant}`}>{this._icon()}</span> {this.dialogTitle}</h3>
            <button type="button" class="bc-close" onClick={() => this._close()}>&times;</button>
          </div>
          <div class="bc-dialog-body">
            {this.message && <p class="bc-dialog-message">{this.message}</p>}
            <slot></slot>
          </div>
          <div class="bc-dialog-footer">
            <button type="button" class={`bc-btn ${btnClass}`} onClick={() => this._close()}>{this.okText}</button>
          </div>
        </div>
      </div>
    );
  }
}
