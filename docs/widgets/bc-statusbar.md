# bc-statusbar

> Status bar showing workflow states

## Quick Start

```html
<bc-statusbar></bc-statusbar>
```

In dark mode, completed and active steps use a brighter progress-like fill, and step separators stay visible so each workflow stage is easy to distinguish.

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| states | string (JSON) | '[]' | Array of state names |
| value | string | '' | Current state |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| setValue(value) | Promise<void> | Set current state |
| getValue() | Promise<string> | Get current state |

See [theming](../theming.md).

