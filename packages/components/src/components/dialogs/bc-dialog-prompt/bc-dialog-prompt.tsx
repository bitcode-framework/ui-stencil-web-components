import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { i18n } from '../../../core/i18n';

@Component({
  tag: 'bc-dialog-prompt',
  styleUrl: 'bc-dialog-prompt.css',
  shadow: false,
})
export class BcDialogPrompt {
  @Element() el!: HTMLElement;
  @Prop({ mutable: true }) open: boolean = false;
  @Prop() dialogTitle: string = '';
  @Prop() message: string = '';
  @Prop() placeholder: string = '';
  @Prop() inputValue: string = '';
  @Prop() okText: string = 'OK';
  @Prop() cancelText: string = 'Cancel';
  @Prop() inputType: 'text' | 'number' | 'password' = 'text';
  @Prop() required: boolean = false;
  @Prop() size: 'sm' | 'md' | 'lg' = 'sm';

  @State() private _value: string = '';

  @Event() lcDialogConfirm!: EventEmitter<{value: string}>;
  @Event() lcDialogCancel!: EventEmitter<void>;
  @Event() lcDialogClose!: EventEmitter<{type: string}>;

  componentWillRender() {
    this.el.dir = i18n.dir;
  }

  @Method() async openDialog(): Promise<void> {
    this._value = this.inputValue || '';
    this.open = true;
    setTimeout(() => {
      const input = this.el.querySelector('input');
      if (input) input.focus();
    }, 50);
  }
  @Method() async closeDialog(): Promise<void> { this._cancel(); }

  private _confirm() {
    if (this.required && !this._value.trim()) return;
    this.open = false;
    this.lcDialogConfirm.emit({value: this._value});
    this.lcDialogClose.emit({type: 'prompt'});
  }

  private _cancel() {
    this.open = false;
    this.lcDialogCancel.emit();
    this.lcDialogClose.emit({type: 'cancel'});
  }

  private _handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter') this._confirm();
    if (e.key === 'Escape') this._cancel();
  }

  render() {
    if (!this.open) return null;
    return (
      <div class="bc-overlay" onClick={() => this._cancel()}>
        <div class={`bc-dialog bc-dialog-${this.size}`} onClick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label={this.dialogTitle}>
          <div class="bc-dialog-header">
            <h3>{this.dialogTitle}</h3>
            <button type="button" class="bc-close" onClick={() => this._cancel()}>&times;</button>
          </div>
          <div class="bc-dialog-body">
            {this.message && <p class="bc-dialog-message">{this.message}</p>}
            <input type={this.inputType} class="bc-prompt-input" value={this._value} placeholder={this.placeholder} required={this.required} onInput={(e: Event) => { this._value = (e.target as HTMLInputElement).value; }} onKeyDown={(e: KeyboardEvent) => this._handleKeyDown(e)} />
            <slot></slot>
          </div>
          <div class="bc-dialog-footer">
            <button type="button" class="bc-btn" onClick={() => this._cancel()}>{this.cancelText}</button>
            <button type="button" class="bc-btn bc-btn-primary" onClick={() => this._confirm()} disabled={this.required && !this._value.trim()}>{this.okText}</button>
          </div>
        </div>
      </div>
    );
  }
}
