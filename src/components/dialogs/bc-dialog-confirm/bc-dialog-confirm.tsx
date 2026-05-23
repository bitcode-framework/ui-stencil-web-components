import { Component, Prop, Event, EventEmitter, Method, h } from '@stencil/core';

@Component({
  tag: 'bc-dialog-confirm',
  styleUrl: 'bc-dialog-confirm.css',
  shadow: false,
})
export class BcDialogConfirm {
  @Prop({ mutable: true }) open: boolean = false;
  @Prop() dialogTitle: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' = 'sm';
  @Prop() message: string = '';
  @Prop() confirmText: string = 'Confirm';
  @Prop() cancelText: string = 'Cancel';
  @Prop() variant: 'default' | 'danger' | 'warning' | 'info' = 'default';

  @Event() lcDialogClose!: EventEmitter<{type: string}>;
  @Event() lcDialogConfirm!: EventEmitter<void>;
  @Event() lcDialogCancel!: EventEmitter<void>;

  @Method() async openDialog(): Promise<void> { this.open = true; }
  @Method() async closeDialog(): Promise<void> { this._cancel(); }

  private _confirm() {
    this.open = false;
    this.lcDialogConfirm.emit();
    this.lcDialogClose.emit({type: 'confirm'});
  }

  private _cancel() {
    this.open = false;
    this.lcDialogCancel.emit();
    this.lcDialogClose.emit({type: 'cancel'});
  }

  render() {
    if (!this.open) return null;
    return (
      <div class="bc-overlay" onClick={() => this._cancel()}>
        <div class={`bc-dialog bc-dialog-${this.size}`} onClick={(e) => e.stopPropagation()} role="alertdialog" aria-modal="true" aria-label={this.dialogTitle}>
          <div class="bc-dialog-header">
            <h3>{this.dialogTitle}</h3>
            <button type="button" class="bc-close" onClick={() => this._cancel()}>&times;</button>
          </div>
          <div class="bc-dialog-body">
            {this.message && <p class="bc-dialog-message">{this.message}</p>}
            <slot></slot>
          </div>
          <div class="bc-dialog-footer">
            <button type="button" class="bc-btn" onClick={() => this._cancel()}>{this.cancelText}</button>
            <button type="button" class={`bc-btn bc-btn-${this.variant === 'danger' ? 'danger' : 'primary'}`} onClick={() => this._confirm()}>{this.confirmText}</button>
          </div>
        </div>
      </div>
    );
  }
}


