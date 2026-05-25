# bc-dialog-alert

> Alert dialog with overlay, OK button, and 5 color variants.

## Quick Start

```html
<bc-dialog-alert
  dialog-title="Success!"
  message="Operation completed successfully."
  variant="success"
></bc-dialog-alert>
```

Open via JavaScript:

```js
document.getElementById('my-alert').open = true;
// or
document.getElementById('my-alert').openDialog();
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| open | boolean | false | Open state (mutable) |
| dialog-title | string | '' | Dialog title |
| message | string | '' | Body message text |
| size | 'sm' \| 'md' \| 'lg' | 'sm' | Dialog size |
| variant | 'default' \| 'success' \| 'danger' \| 'warning' \| 'info' | 'default' | Color variant — affects icon and OK button style |
| ok-text | string | 'OK' | OK button label |

## Variants

| Variant | Icon | OK Button |
|---------|------|-----------|
| `default` | 💬 | Primary (purple) |
| `success` | ✓ | Green |
| `danger` | ✕ | Red |
| `warning` | ⚠ | Yellow/amber |
| `info` | ℹ | Blue |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcDialogClose | `{ type: string }` | Dialog closed (`type` is 'alert') |
| lcDialogConfirm | void | OK button clicked |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| openDialog() | Promise\<void\> | Open the dialog |
| closeDialog() | Promise\<void\> | Close the dialog |

## Slots

| Slot | Description |
|------|-------------|
| default | Custom body content (rendered below message) |

## Examples

### Default

```html
<bc-dialog-alert id="a1" dialog-title="Notice" message="This is a default alert."></bc-dialog-alert>
<button onclick="document.getElementById('a1').open=true">Default</button>
```

### Success

```html
<bc-dialog-alert id="a2" dialog-title="Success!" message="Operation completed successfully." variant="success"></bc-dialog-alert>
<button onclick="document.getElementById('a2').open=true">Success</button>
```

### Danger

```html
<bc-dialog-alert id="a3" dialog-title="Error" message="Something went wrong." variant="danger"></bc-dialog-alert>
<button onclick="document.getElementById('a3').open=true">Error</button>
```

### Warning

```html
<bc-dialog-alert id="a4" dialog-title="Warning" message="Please review before proceeding." variant="warning"></bc-dialog-alert>
<button onclick="document.getElementById('a4').open=true">Warning</button>
```

### Info

```html
<bc-dialog-alert id="a5" dialog-title="Information" message="Here is some useful information." variant="info"></bc-dialog-alert>
<button onclick="document.getElementById('a5').open=true">Info</button>
```

See [theming](../theming.md).
