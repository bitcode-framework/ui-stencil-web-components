import { applyChartTheme } from './chart-theme';

describe('applyChartTheme', () => {
  beforeEach(() => {
    document.documentElement.setAttribute('data-bc-theme', 'dark');
    document.documentElement.style.setProperty('--bc-text', '#f1f5f9');
    document.documentElement.style.setProperty('--bc-text-secondary', '#94a3b8');
    document.documentElement.style.setProperty('--bc-border-color', '#334155');
    document.documentElement.style.setProperty('--bc-bg-tertiary', '#475569');
    document.documentElement.style.setProperty('--bc-bg-secondary', '#1e293b');
    document.documentElement.style.setProperty('--bc-primary', '#818cf8');
  });

  afterEach(() => {
    document.documentElement.removeAttribute('data-bc-theme');
    document.documentElement.removeAttribute('style');
  });

  it('applies themed colors to axes and title', () => {
    const themed = applyChartTheme({
      title: { text: 'Sales' },
      xAxis: { type: 'category', axisLabel: {} },
      yAxis: { type: 'value' },
    }, '');

    expect(themed.darkMode).toBe(true);
    expect((themed.title as Record<string, unknown>).textStyle).toMatchObject({ color: '#f1f5f9' });
    expect((themed.xAxis as Record<string, unknown>).axisLabel).toMatchObject({ color: '#94a3b8' });
    expect((themed.yAxis as Record<string, unknown>).axisLine).toMatchObject({
      lineStyle: { color: '#334155' },
    });
  });

  it('applies themed colors to parallel axes and series labels', () => {
    const themed = applyChartTheme({
      parallelAxis: [{ dim: 0, name: 'Price' }],
      series: [{ type: 'parallel', label: { show: true } }],
    }, '');

    expect((themed.parallelAxis as Array<Record<string, unknown>>)[0].nameTextStyle).toMatchObject({ color: '#94a3b8' });
    expect((themed.series as Array<Record<string, unknown>>)[0].label).toMatchObject({ color: '#f1f5f9' });
  });
});
