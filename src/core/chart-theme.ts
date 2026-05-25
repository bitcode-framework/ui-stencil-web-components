import { BcSetup } from './bc-setup';

interface ChartThemeTokens {
  text: string;
  textSecondary: string;
  border: string;
  grid: string;
  surface: string;
  primary: string;
}

function readThemeToken(name: string, fallback: string): string {
  if (typeof document === 'undefined') return fallback;
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  return value || fallback;
}

function getChartThemeTokens(resolvedTheme: string): ChartThemeTokens {
  const isDark = resolvedTheme === 'dark';
  return {
    text: readThemeToken('--bc-text', isDark ? '#f1f5f9' : '#111827'),
    textSecondary: readThemeToken('--bc-text-secondary', isDark ? '#94a3b8' : '#6b7280'),
    border: readThemeToken('--bc-border-color', isDark ? '#334155' : '#d1d5db'),
    grid: readThemeToken('--bc-bg-tertiary', isDark ? '#475569' : '#e5e7eb'),
    surface: readThemeToken('--bc-bg-secondary', isDark ? '#1e293b' : '#ffffff'),
    primary: readThemeToken('--bc-primary', isDark ? '#818cf8' : '#4f46e5'),
  };
}

export function resolveChartTheme(theme: string): string {
  if (theme === 'dark' || theme === 'light') return theme;
  if (theme) return theme;
  if (typeof document === 'undefined') return 'light';
  return document.documentElement.getAttribute('data-bc-theme') || BcSetup.getResolvedTheme();
}

function toObjectArray(value: unknown): Record<string, unknown>[] {
  if (Array.isArray(value)) {
    return value.filter((item): item is Record<string, unknown> => item !== null && typeof item === 'object');
  }
  if (value !== null && typeof value === 'object') {
    return [value as Record<string, unknown>];
  }
  return [];
}

function withNestedObject(base: unknown, key: string, value: Record<string, unknown>): Record<string, unknown> {
  const current = (base !== null && typeof base === 'object') ? base as Record<string, unknown> : {};
  const nested = (current[key] !== null && typeof current[key] === 'object') ? current[key] as Record<string, unknown> : {};
  return {
    ...current,
    [key]: {
      ...nested,
      ...value,
    },
  };
}

function buildAxisThemePatch(axis: Record<string, unknown>, tokens: ChartThemeTokens): Record<string, unknown> {
  return {
    ...axis,
    axisLine: withNestedObject(axis.axisLine, 'lineStyle', { color: tokens.border }),
    axisTick: withNestedObject(axis.axisTick, 'lineStyle', { color: tokens.border }),
    axisLabel: {
      ...((axis.axisLabel !== null && typeof axis.axisLabel === 'object') ? axis.axisLabel as Record<string, unknown> : {}),
      color: tokens.textSecondary,
    },
    nameTextStyle: {
      ...((axis.nameTextStyle !== null && typeof axis.nameTextStyle === 'object') ? axis.nameTextStyle as Record<string, unknown> : {}),
      color: tokens.textSecondary,
    },
    splitLine: withNestedObject(axis.splitLine, 'lineStyle', { color: tokens.grid }),
    splitArea: withNestedObject(axis.splitArea, 'areaStyle', { color: [tokens.surface, 'transparent'] }),
  };
}

function buildSeriesThemePatch(series: Record<string, unknown>, tokens: ChartThemeTokens): Record<string, unknown> {
  return {
    ...series,
    label: {
      ...((series.label !== null && typeof series.label === 'object') ? series.label as Record<string, unknown> : {}),
      color: tokens.text,
    },
    endLabel: {
      ...((series.endLabel !== null && typeof series.endLabel === 'object') ? series.endLabel as Record<string, unknown> : {}),
      color: tokens.text,
    },
    upperLabel: {
      ...((series.upperLabel !== null && typeof series.upperLabel === 'object') ? series.upperLabel as Record<string, unknown> : {}),
      color: tokens.text,
    },
    title: {
      ...((series.title !== null && typeof series.title === 'object') ? series.title as Record<string, unknown> : {}),
      color: tokens.textSecondary,
    },
    detail: {
      ...((series.detail !== null && typeof series.detail === 'object') ? series.detail as Record<string, unknown> : {}),
      color: tokens.text,
    },
    markLine: withNestedObject(series.markLine, 'label', { color: tokens.textSecondary }),
    markPoint: withNestedObject(series.markPoint, 'label', { color: tokens.text }),
    markArea: withNestedObject(series.markArea, 'label', { color: tokens.text }),
  };
}

export function applyChartTheme(option: Record<string, unknown>, theme: string): Record<string, unknown> {
  const resolvedTheme = resolveChartTheme(theme);
  const tokens = getChartThemeTokens(resolvedTheme);
  const title = toObjectArray(option.title).map(entry => ({
    ...entry,
    textStyle: {
      ...((entry.textStyle !== null && typeof entry.textStyle === 'object') ? entry.textStyle as Record<string, unknown> : {}),
      color: tokens.text,
    },
    subtextStyle: {
      ...((entry.subtextStyle !== null && typeof entry.subtextStyle === 'object') ? entry.subtextStyle as Record<string, unknown> : {}),
      color: tokens.textSecondary,
    },
  }));
  const legend = toObjectArray(option.legend).map(entry => ({
    ...entry,
    textStyle: {
      ...((entry.textStyle !== null && typeof entry.textStyle === 'object') ? entry.textStyle as Record<string, unknown> : {}),
      color: tokens.textSecondary,
    },
  }));
  const toolbox = toObjectArray(option.toolbox).map(entry => ({
    ...entry,
    iconStyle: {
      ...((entry.iconStyle !== null && typeof entry.iconStyle === 'object') ? entry.iconStyle as Record<string, unknown> : {}),
      borderColor: tokens.textSecondary,
    },
    emphasis: withNestedObject(entry.emphasis, 'iconStyle', { borderColor: tokens.primary }),
  }));
  const visualMap = toObjectArray(option.visualMap).map(entry => ({
    ...entry,
    textStyle: {
      ...((entry.textStyle !== null && typeof entry.textStyle === 'object') ? entry.textStyle as Record<string, unknown> : {}),
      color: tokens.textSecondary,
    },
  }));
  const radar = toObjectArray(option.radar).map(entry => ({
    ...entry,
    name: withNestedObject(entry.name, 'textStyle', { color: tokens.textSecondary }),
    axisLine: withNestedObject(entry.axisLine, 'lineStyle', { color: tokens.border }),
    splitLine: withNestedObject(entry.splitLine, 'lineStyle', { color: tokens.grid }),
  }));
  const calendar = toObjectArray(option.calendar).map(entry => ({
    ...entry,
    itemStyle: {
      ...((entry.itemStyle !== null && typeof entry.itemStyle === 'object') ? entry.itemStyle as Record<string, unknown> : {}),
      borderColor: tokens.border,
    },
    dayLabel: {
      ...((entry.dayLabel !== null && typeof entry.dayLabel === 'object') ? entry.dayLabel as Record<string, unknown> : {}),
      color: tokens.textSecondary,
    },
    monthLabel: {
      ...((entry.monthLabel !== null && typeof entry.monthLabel === 'object') ? entry.monthLabel as Record<string, unknown> : {}),
      color: tokens.textSecondary,
    },
    yearLabel: {
      ...((entry.yearLabel !== null && typeof entry.yearLabel === 'object') ? entry.yearLabel as Record<string, unknown> : {}),
      color: tokens.textSecondary,
    },
  }));
  const nextOption: Record<string, unknown> = {
    ...option,
    darkMode: resolvedTheme === 'dark',
    textStyle: {
      ...((option.textStyle !== null && typeof option.textStyle === 'object') ? option.textStyle as Record<string, unknown> : {}),
      color: tokens.text,
    },
  };

  if (title.length > 0) nextOption.title = Array.isArray(option.title) ? title : title[0];
  if (legend.length > 0) nextOption.legend = Array.isArray(option.legend) ? legend : legend[0];
  if (toolbox.length > 0) nextOption.toolbox = Array.isArray(option.toolbox) ? toolbox : toolbox[0];
  if (visualMap.length > 0) nextOption.visualMap = Array.isArray(option.visualMap) ? visualMap : visualMap[0];
  if (radar.length > 0) nextOption.radar = Array.isArray(option.radar) ? radar : radar[0];
  if (calendar.length > 0) nextOption.calendar = Array.isArray(option.calendar) ? calendar : calendar[0];

  const axisKeys = ['xAxis', 'yAxis', 'angleAxis', 'radiusAxis', 'singleAxis', 'parallelAxis'];
  for (const key of axisKeys) {
    const axes = toObjectArray(option[key]);
    if (axes.length > 0) {
      const themedAxes = axes.map(axis => buildAxisThemePatch(axis, tokens));
      nextOption[key] = Array.isArray(option[key]) ? themedAxes : themedAxes[0];
    }
  }

  const series = toObjectArray(option.series);
  if (series.length > 0) {
    const themedSeries = series.map(entry => buildSeriesThemePatch(entry, tokens));
    nextOption.series = Array.isArray(option.series) ? themedSeries : themedSeries[0];
  }

  return nextOption;
}
