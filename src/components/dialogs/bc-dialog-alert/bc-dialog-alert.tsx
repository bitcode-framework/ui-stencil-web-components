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

  render() {
    if (!this.open) return null;
    return (
      <div class="bc-overlay" onClick={() => this._close()}>
        <div class={`bc-dialog bc-dialog-${this.size}`} onClick={(e) => e.stopPropagation()} role="alertdialog" aria-modal="true" aria-label={this.dialogTitle}>
          <div class="bc-dialog-header">
            <h3>{this.dialogTitle}</h3>
            <button type="button" class="bc-close" onClick={() => this._close()}>&times;</button>
          </div>
          <div class="bc-dialog-body">
            {this.message && <p class="bc-dialog-message">{this.message}</p>}
            <slot></slot>
          </div>
          <div class="bc-dialog-footer">
            <button type="button" class="bc-btn bc-btn-primary" onClick={() => this._close()}>{this.okText}</button>
          </div>
        </div>
      </div>
    );
  }
}
