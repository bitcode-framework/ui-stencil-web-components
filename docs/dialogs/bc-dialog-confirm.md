# bc-dialog-confirm

> Confirmation dialog with overlay, cancel + confirm buttons, and 4 color variants.

## Quick Start

```html
<bc-dialog-confirm
  dialog-title="Delete?"
  message="This cannot be undone."
  variant="danger"
  confirm-text="Delete"
  cancel-text="Cancel"
></bc-dialog-confirm>
```

Open via JavaScript:

```js
document.getElementById('my-confirm').open = true;
// or
document.getElementById('my-confirm').openDialog();
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| open | boolean | false | Open state (mutable) |
| dialog-title | string | '' | Dialog title |
| message | string | '' | Body message text |
| size | 'sm' \| 'md' \| 'lg' | 'sm' | Dialog size |
| variant | 'default' \| 'danger' \| 'warning' \| 'info' | 'default' | Color variant — affects icon and confirm button style |
| confirm-text | string | 'Confirm' | Confirm button label |
| cancel-text | string | 'Cancel' | Cancel button label |

## Variants

| Variant | Icon | Confirm Button |
|---------|------|----------------|
| `default` | ❓ | Primary (purple) |
| `danger` | ⚠ | Red |
| `warning` | ⚡ | Yellow/amber |
| `info` | ℹ | Blue |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcDialogClose | `{ type: string }` | Dialog closed (`type` is 'confirm' or 'cancel') |
| lcDialogConfirm | void | Confirm button clicked |
| lcDialogCancel | void | Cancel button or overlay clicked |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| openDialog() | Promise\<void\> | Open the dialog |
| closeDialog() | Promise\<void\> | Close (cancels) |

## Slots

| Slot | Description |
|------|-------------|
| default | Custom body content (rendered below message) |

## Examples

### Default

```html
<bc-dialog-confirm id="c1" dialog-title="Confirm" message="Are you sure?" confirm-text="Yes" cancel-text="No"></bc-dialog-confirm>
<button onclick="document.getElementById('c1').open=true">Default</button>
```

### Danger

```html
<bc-dialog-confirm id="c2" dialog-title="Delete?" message="This cannot be undone." variant="danger" confirm-text="Delete" cancel-text="Cancel"></bc-dialog-confirm>
<button onclick="document.getElementById('c2').open=true">Delete</button>
```

### Warning

```html
<bc-dialog-confirm id="c3" dialog-title="Warning" message="Proceed with caution." variant="warning" confirm-text="Continue" cancel-text="Go Back"></bc-dialog-confirm>
<button onclick="document.getElementById('c3').open=true">Warning</button>
```

### Info

```html
<bc-dialog-confirm id="c4" dialog-title="Information" message="This action requires confirmation." variant="info" confirm-text="Got it" cancel-text="Cancel"></bc-dialog-confirm>
<button onclick="document.getElementById('c4').open=true">Info</button>
```

See [theming](../theming.md).
