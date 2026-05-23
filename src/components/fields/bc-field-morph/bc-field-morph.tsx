import { Component, Prop, State, Event, EventEmitter, Method, Element, h } from '@stencil/core';
import { FieldChangeEvent, FieldFocusEvent, FieldBlurEvent, FieldClearEvent, FieldValidationEvent, FieldValidEvent, ValidationResult, ValidateOn } from '../../../core/types';
import { FieldState, createFieldState, markDirty, markTouched, getFieldClasses, validateFieldValue } from '../../../core/field-utils';
import { BcSetup } from '../../../core/bc-setup';
import { fetchOptions } from '../../../core/data-fetcher';

@Component({ tag: 'bc-field-morph', styleUrl: 'bc-field-morph.css', shadow: false })
export class BcFieldMorph {
  @Element() el!: HTMLElement;
  @Prop() name: string = '';
  @Prop() label: string = '';
  @Prop({ mutable: true }) value: string = '';
  @Prop({ mutable: true }) morphType: string = '';
  @Prop() placeholder: string = 'Search...';
  @Prop() models: string = '[]';
  @Prop() required: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() disabled: boolean = false;

  @Prop({ mutable: true }) validationStatus: 'none' | 'validating' | 'valid' | 'invalid' = 'none';
  @Prop({ mutable: true }) validationMessage: string = '';
  @Prop() hint: string = '';
  @Prop() size: 'sm' | 'md' | 'lg' = 'md';
  @Prop() clearable: boolean = true;
  @Prop() tooltip: string = '';
  @Prop() loading: boolean = false;
  @Prop() defaultValue: string = '';
  @Prop() validateOn: ValidateOn | '' = '';

  @State() query: string = '';
  @State() results: Array<Record<string, unknown>> = [];
  @State() showDropdown: boolean = false;
  @State() private _fieldState: FieldState = createFieldState('');

  private debounceTimer: ReturnType<typeof setTimeout> | null = null;
  customValidator?: (value: unknown) => string | null | Promise<string | null>;

  @Event() lcFieldChange!: EventEmitter<FieldChangeEvent>;
  @Event() lcFieldFocus!: EventEmitter<FieldFocusEvent>;
  @Event() lcFieldBlur!: EventEmitter<FieldBlurEvent>;
  @Event() lcFieldClear!: EventEmitter<FieldClearEvent>;
  @Event() lcFieldInvalid!: EventEmitter<FieldValidationEvent>;
  @Event() lcFieldValid!: EventEmitter<FieldValidEvent>;

  componentWillLoad() {
    this._fieldState = createFieldState(this.value || this.defaultValue);
    if (!this.value && this.defaultValue) this.value = this.defaultValue;
  }

  private _getValidateOn(): ValidateOn { return (this.validateOn as ValidateOn) || BcSetup.getConfig().validateOn || 'blur'; }

  private getModelList(): string[] {
    try { return JSON.parse(this.models); }
    catch { return []; }
  }

  private handleTypeChange(e: Event) {
    const newType = (e.target as HTMLSelectElement).value;
    const old = this.value;
    this.morphType = newType;
    this.value = '';
    this.query = '';
    this.results = [];
    this._fieldState = markDirty(this._fieldState, '');
    this.lcFieldChange.emit({ name: this.name, value: '', oldValue: old });
  }

  private async search(q: string) {
    this.query = q;
    if (this.debounceTimer) clearTimeout(this.debounceTimer);
    if (q.length < 1 || !this.morphType) { this.results = []; this.showDropdown = false; return; }
    this.debounceTimer = setTimeout(async () => {
      try {
        this.results = await fetchOptions({ element: this.el, model: this.morphType, query: q }) as Array<Record<string, unknown>>;
        this.showDropdown = this.results.length > 0;
      } catch { this.results = []; this.showDropdown = false; }
    }, 300);
  }

  private select(item: Record<string, unknown>) {
    const old = this.value;
    this.value = String(item['id'] || '');
    this.showDropdown = false;
    this._fieldState = markDirty(this._fieldState, this.value);
    this.lcFieldChange.emit({ name: this.name, value: this.value, oldValue: old });
  }

  private handleClear() {
    const old = this.value;
    this.value = '';
    this.morphType = '';
    this.query = '';
    this._fieldState = markDirty(this._fieldState, '');
    this.lcFieldClear.emit({ name: this.name, oldValue: old });
    this.lcFieldChange.emit({ name: this.name, value: '', oldValue: old });
  }

  private handleFocus() { this.lcFieldFocus.emit({ name: this.name, value: this.value }); }
  private handleBlur() {
    this._fieldState = markTouched(this._fieldState);
    this.lcFieldBlur.emit({ name: this.name, value: this.value, dirty: this._fieldState.dirty, touched: true });
    if (this._getValidateOn() === 'blur') this._runValidation();
  }

  private async _runValidation(): Promise<ValidationResult> {
    this.validationStatus = 'validating';
    const result = await validateFieldValue(this.value, { required: this.required }, { customValidator: this.customValidator });
    if (result.valid) { this.validationStatus = 'valid'; this.validationMessage = ''; this.lcFieldValid.emit({ name: this.name, value: this.value }); }
    else { this.validationStatus = 'invalid'; this.validationMessage = result.errors[0] || ''; this.lcFieldInvalid.emit({ name: this.name, value: this.value, errors: result.errors }); }
    return result;
  }

  @Method() async validate(): Promise<ValidationResult> { return this._runValidation(); }
  @Method() async reset(): Promise<void> { this.value = this._fieldState.initialValue as string || this.defaultValue || ''; this.morphType = ''; this._fieldState = createFieldState(this.value); this.validationStatus = 'none'; this.validationMessage = ''; }
  @Method() async clear(): Promise<void> { this.handleClear(); }
  @Method() async setValue(value: string, emit: boolean = true): Promise<void> { const old = this.value; this.value = value; this._fieldState = markDirty(this._fieldState, value); if (emit) this.lcFieldChange.emit({ name: this.name, value, oldValue: old }); }
  @Method() async getValue(): Promise<string> { return this.value; }
  @Method() async focusField(): Promise<void> { this.el.querySelector('input')?.focus(); }
  @Method() async blurField(): Promise<void> { this.el.querySelector('input')?.blur(); }
  @Method() async isDirty(): Promise<boolean> { return this._fieldState.dirty; }
  @Method() async isTouched(): Promise<boolean> { return this._fieldState.touched; }
  @Method() async setError(message: string): Promise<void> { this.validationStatus = 'invalid'; this.validationMessage = message; }
  @Method() async clearError(): Promise<void> { this.validationStatus = 'none'; this.validationMessage = ''; }

  render() {
    const fieldClasses = getFieldClasses({ size: this.size, validationStatus: this.validationStatus, disabled: this.disabled, readonly: this.readonly, loading: this.loading, dirty: this._fieldState.dirty, touched: this._fieldState.touched });
    const showError = this.validationStatus === 'invalid' && this.validationMessage;
    const showHint = this.hint && !showError;
    const modelList = this.getModelList();

    return (
      <div class={{ ...fieldClasses, 'bc-morph-wrap': true }}>
        {this.label && <label class="bc-field-label">{this.label}{this.required && <span class="required">*</span>}{this.tooltip && <span class="bc-field-tooltip" title={this.tooltip}>?</span>}</label>}
        <div class="bc-morph-selectors">
          <select class="bc-field-input bc-morph-type-select" disabled={this.disabled || this.readonly} onChange={(e) => this.handleTypeChange(e)}>
            <option value="">Select type...</option>
            {modelList.map(m => <option value={m} selected={this.morphType === m}>{m}</option>)}
          </select>
          <div class="bc-morph-record-wrap">
            <input type="text" class="bc-field-input" placeholder={this.morphType ? this.placeholder : 'Select type first'} readOnly={this.readonly} disabled={this.disabled || !this.morphType} value={this.value || this.query} onInput={(e: Event) => this.search((e.target as HTMLInputElement).value)} onFocus={() => this.handleFocus()} onBlur={() => { setTimeout(() => { this.showDropdown = false; }, 200); this.handleBlur(); }} />
            {this.clearable && this.value && !this.disabled && !this.readonly && <button type="button" class="bc-field-clear-btn" onClick={() => this.handleClear()} tabIndex={-1}>&times;</button>}
            {this.loading && <span class="bc-field-loading-indicator" />}
            {this.showDropdown && (
              <div class="bc-morph-dropdown">
                {this.results.map(item => <div class="bc-morph-option" onMouseDown={() => this.select(item)}>{String(item['name'] || item['title'] || item['label'] || item['id'] || '')}</div>)}
              </div>
            )}
          </div>
        </div>
        <input type="hidden" name={this.name + '_type'} value={this.morphType} />
        <input type="hidden" name={this.name + '_id'} value={this.value} />
        <div class="bc-field-footer">
          {showError && <div class="bc-field-error" role="alert">{this.validationMessage}</div>}
          {showHint && <div class="bc-field-hint">{this.hint}</div>}
        </div>
      </div>
    );
  }
}
