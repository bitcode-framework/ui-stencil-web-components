import { EventEmitter, VNode, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from './types';
import { FieldState, createFieldState, markDirty, markTouched, validateFieldValue, debounce } from './field-utils';
import { BcSetup } from './bc-setup';

export interface FieldBaseConfig<V> {
  defaultValue: V;
  emptyValue: V;
  extraValidation?: (value: V) => Record<string, unknown>;
}

export interface FieldHost<V> {
  el: HTMLElement;
  name: string;
  value: V;
  required: boolean;
  readonly: boolean;
  disabled: boolean;
  validationStatus: 'none' | 'validating' | 'valid' | 'invalid';
  validationMessage: string;
  hint: string;
  size: 'sm' | 'md' | 'lg';
  clearable: boolean;
  tooltip: string;
  loading: boolean;
  autofocus: boolean;
  defaultValue: V;
  validateOn: ValidateOn | '';
  dependOn?: string;
  lcFieldChange: EventEmitter<FieldChangeEvent>;
  lcFieldFocus: EventEmitter<FieldFocusEvent>;
  lcFieldBlur: EventEmitter<FieldBlurEvent>;
  lcFieldClear: EventEmitter<FieldClearEvent>;
  lcFieldInvalid: EventEmitter<FieldValidationEvent>;
  lcFieldValid: EventEmitter<FieldValidEvent>;
}

export class FieldBase<V> {
  private host: FieldHost<V>;
  private config: FieldBaseConfig<V>;
  private _fieldState: FieldState;
  private _dependListener?: (e: Event) => void;

  customValidator?: (value: unknown) => string | null | Promise<string | null>;
  validators?: Array<{ rule: string | ((value: unknown) => boolean | Promise<boolean>); message: string }>;
  serverValidator?: string | ((value: unknown) => Promise<string | null>);

  constructor(host: FieldHost<V>, config: FieldBaseConfig<V>) {
    this.host = host;
    this.config = config;
    this._fieldState = createFieldState(config.defaultValue);
  }

  get fieldState(): FieldState { return this._fieldState; }

  init(value: V): V {
    this._fieldState = createFieldState(value || this.config.defaultValue);
    return value || this.config.defaultValue;
  }

  setValue(value: V, emit: boolean = true): void {
    const oldValue = this.host.value;
    this.host.value = value;
    this._fieldState = markDirty(this._fieldState, value);
    if (emit) {
      this.host.lcFieldChange.emit({ name: this.host.name, value, oldValue });
    }
  }

  clearValue(inputEl?: HTMLElement): void {
    const oldValue = this.host.value;
    this.host.value = this.config.emptyValue;
    this._fieldState = markDirty(this._fieldState, this.config.emptyValue);
    this.host.lcFieldClear.emit({ name: this.host.name, oldValue });
    this.host.lcFieldChange.emit({ name: this.host.name, value: this.config.emptyValue, oldValue });
    inputEl?.focus();
  }

  resetValue(): void {
    this.host.value = this._fieldState.initialValue as V || this.config.defaultValue;
    this._fieldState = createFieldState(this.host.value);
    this.host.validationStatus = 'none';
    this.host.validationMessage = '';
  }

  emitFocus(): void {
    this.host.lcFieldFocus.emit({ name: this.host.name, value: this.host.value });
  }

  emitBlur(): void {
    this._fieldState = markTouched(this._fieldState);
    this.host.lcFieldBlur.emit({
      name: this.host.name,
      value: this.host.value,
      dirty: this._fieldState.dirty,
      touched: true,
    });
    if (this.getValidateOn() === 'blur') {
      this.runValidation();
    }
  }

  handleChange(emit: boolean = true): void {
    this._fieldState = markDirty(this._fieldState, this.host.value);
    if (emit) {
      this.host.lcFieldChange.emit({ name: this.host.name, value: this.host.value, oldValue: undefined });
    }
    if (this.getValidateOn() === 'change') {
      debounce(`validate-${this.host.name}`, () => this.runValidation(), 300);
    }
  }

  getValidateOn(): ValidateOn {
    return (this.host.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur';
  }

  async runValidation(): Promise<ValidationResult> {
    this.host.validationStatus = 'validating';
    const extra = this.config.extraValidation ? this.config.extraValidation(this.host.value) : {};
    const result = await validateFieldValue(
      this.host.value,
      {
        required: this.host.required,
        ...extra,
      },
      {
        validators: this.validators,
        customValidator: this.customValidator,
        serverValidator: this.serverValidator,
      },
    );

    if (result.valid) {
      this.host.validationStatus = 'valid';
      this.host.validationMessage = '';
      this.host.lcFieldValid.emit({ name: this.host.name, value: this.host.value });
    } else {
      this.host.validationStatus = 'invalid';
      this.host.validationMessage = result.errors[0] || '';
      this.host.lcFieldInvalid.emit({ name: this.host.name, value: this.host.value, errors: result.errors });
    }
    return result;
  }

  setupDependencyListener(): void {
    if (!this.host.dependOn) return;
    this._dependListener = (e: Event) => {
      const detail = (e as CustomEvent<FieldChangeEvent>).detail;
      if (!detail) return;
      const deps = this.host.dependOn!.split(',').map(d => d.trim());
      if (deps.includes(detail.name)) {
        this.host.value = this.config.emptyValue;
        this._fieldState = createFieldState(this.config.emptyValue);
        this.host.lcFieldChange.emit({ name: this.host.name, value: this.config.emptyValue, oldValue: detail.value });
      }
    };
    document.addEventListener('lcFieldChange', this._dependListener);
  }

  cleanupDependencyListener(): void {
    if (this._dependListener) {
      document.removeEventListener('lcFieldChange', this._dependListener);
      this._dependListener = undefined;
    }
  }

  setValidationStatus(valid: boolean, errors: string[]): void {
    if (valid) {
      this.host.validationStatus = 'valid';
      this.host.validationMessage = '';
      this.host.lcFieldValid.emit({ name: this.host.name, value: this.host.value });
    } else {
      this.host.validationStatus = 'invalid';
      this.host.validationMessage = errors[0] || '';
      this.host.lcFieldInvalid.emit({ name: this.host.name, value: this.host.value, errors });
    }
  }
}

export function renderLabel(host: { name: string; label: string; required: boolean; tooltip: string }): VNode | null {
  if (!host.label) return null;
  const children: Array<string | VNode> = [host.label];
  if (host.required) children.push(h('span', { class: { 'required': true } }, '*'));
  if (host.tooltip) children.push(h('span', { class: { 'bc-field-tooltip': true }, title: host.tooltip }, '?'));
  return h('label', { class: { 'bc-field-label': true }, htmlFor: host.name }, ...children);
}

export function renderFooter(host: { name: string; validationStatus: string; validationMessage: string; hint: string }): VNode {
  const showError = host.validationStatus === 'invalid' && host.validationMessage;
  const showHint = host.hint && !showError;
  const children: Array<VNode> = [];
  if (showError) children.push(h('div', { class: { 'bc-field-error': true }, id: `${host.name}-error`, role: 'alert' }, host.validationMessage));
  if (showHint) children.push(h('div', { class: { 'bc-field-hint': true }, id: `${host.name}-hint` }, host.hint));
  return h('div', { class: { 'bc-field-footer': true } }, ...children);
}
