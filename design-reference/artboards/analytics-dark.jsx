// Analytics — Direction A (analytics-dark.jsx)
function AnalyticsDark() {
  const W = 1440, H = 1100;
  return (
    <div style={{ width: W, height: H, background: '#0a0a0a', color: '#f4f4f0', fontFamily: 'var(--font-sans)', display: 'flex' }}>
      <Sidebar active="analytics"/>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {/* header: breadcrumb "Analytics / Conversion funnel" + period segmented [7d 30d(active) 90d YTD All] + Compare + Export CSV */}
        <div style={{ height: 56, borderBottom: '1px solid #17171a', display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 24px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <span style={{ color: '#a3a39c', fontSize: 13 }}>Analytics</span>
            <span style={{ color: '#3a3a40' }}>/</span>
            <span style={{ fontSize: 14, fontWeight: 500 }}>Conversion funnel</span>
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <div style={{ display: 'flex', background: '#111110', border: '1px solid #232326', borderRadius: 8, padding: 2 }}>
              {['7d', '30d', '90d', 'YTD', 'All'].map((t, i) => (
                <span key={t} style={{ padding: '5px 12px', fontSize: 12, borderRadius: 6, background: i === 1 ? '#17171a' : 'transparent', color: i === 1 ? '#f4f4f0' : '#a3a39c' }}>{t}</span>
              ))}
            </div>
            <button className="btn-ghost" style={{ padding: '7px 12px', fontSize: 12 }}>Compare</button>
            <button className="btn-ghost" style={{ padding: '7px 12px', fontSize: 12 }}>Export CSV</button>
          </div>
        </div>

        <div className="zz-scroll" style={{ flex: 1, overflowY: 'auto', padding: 24 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', marginBottom: 24 }}>
            <div>
              <span className="mono tracked" style={{ fontSize: 10, color: '#66665f' }}>1 APR — 28 APR 2026</span>
              <h1 style={{ fontSize: 32, fontWeight: 500, letterSpacing: '-0.02em', margin: '6px 0 0' }}>Performance overview</h1>
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <Pill tone="neutral">All workflows</Pill>
              <Pill tone="neutral">All accounts</Pill>
            </div>
          </div>

          {/* Big chart card: row of Big metrics + legend, then BigChart */}
          <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 20, marginBottom: 12 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 18 }}>
              <div style={{ display: 'flex', gap: 28 }}>
                <Big label="COMMENTS" v="38,247" d="+22%" color={ZZ_LIME}/>
                <Big label="DMs SENT" v="26,891" d="+18%" color="oklch(0.78 0.16 240)"/>
                <Big label="LEADS" v="6,142" d="+34%" color="oklch(0.78 0.2 0)"/>
                <Big label="CVR%" v="22.8%" d="+4.2pt" color="oklch(0.85 0.16 75)"/>
                <Big label="EST. REVENUE" v="Rp 348jt" d="+41%" color={ZZ_LIME}/>
              </div>
              <div style={{ display: 'flex', gap: 12, fontSize: 11, color: '#a3a39c' }}>
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><span style={{ width: 8, height: 2, background: ZZ_LIME }}/>Comments</span>
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><span style={{ width: 8, height: 2, background: 'oklch(0.78 0.16 240)' }}/>DMs</span>
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><span style={{ width: 8, height: 2, background: 'oklch(0.78 0.2 0)' }}/>Leads</span>
              </div>
            </div>
            <BigChart/>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1.4fr 1fr', gap: 12 }}>
            {/* Funnel comment-to-cash: each step → index 01.., label, n (right), p% (right), bar (w280 h22, color per step, opacity 0.9), drop% (pink) */}
            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 20 }}>
              <h3 style={{ fontSize: 16, fontWeight: 500, margin: '0 0 16px' }}>Funnel comment-to-cash</h3>
              {[
                { l: 'Comments received',  n: 38247, p: 1.0,    c: ZZ_LIME },
                { l: 'Passed filter',      n: 31204, p: 0.815,  c: ZZ_LIME, drop: '−18.4%' },
                { l: 'Comment replied',    n: 30891, p: 0.807,  c: 'oklch(0.78 0.2 0)', drop: '−1.0%' },
                { l: 'DM delivered',       n: 26891, p: 0.703,  c: 'oklch(0.78 0.2 0)', drop: '−12.9%' },
                { l: 'AI conversation',    n: 14207, p: 0.371,  c: 'oklch(0.78 0.16 240)', drop: '−47.2%' },
                { l: 'Leads captured',     n: 6142,  p: 0.160,  c: ZZ_LIME, drop: '−56.7%' },
                { l: 'Purchases (tracked)',n: 1402,  p: 0.0367, c: 'oklch(0.85 0.16 75)', drop: '−77.2%' },
              ].map((s, i) => (
                <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8 }}>
                  <span className="mono" style={{ fontSize: 10.5, color: '#66665f', width: 22 }}>{String(i+1).padStart(2, '0')}</span>
                  <span style={{ flex: 1, fontSize: 13 }}>{s.l}</span>
                  <span className="mono tnum" style={{ fontSize: 12, width: 80, textAlign: 'right' }}>{s.n.toLocaleString()}</span>
                  <span className="mono tnum" style={{ fontSize: 11, color: '#66665f', width: 44, textAlign: 'right' }}>{(s.p * 100).toFixed(1)}%</span>
                  <div style={{ width: 280, height: 22, background: '#17171a', borderRadius: 4, position: 'relative' }}>
                    <div style={{ width: `${s.p * 100}%`, height: 22, background: s.c, opacity: 0.9, borderRadius: 4 }}/>
                  </div>
                  <span className="mono" style={{ fontSize: 10.5, color: s.drop ? 'oklch(0.78 0.2 0)' : '#66665f', width: 50, textAlign: 'right' }}>{s.drop || ''}</span>
                </div>
              ))}
              {/* AI insight callout (warn icon I.sparkle) + "Apply suggestion" link */}
              <div style={{ marginTop: 14, padding: 12, background: '#17171a', borderRadius: 8, display: 'flex', alignItems: 'center', gap: 10 }}>
                <span style={{ width: 28, height: 28, background: 'oklch(0.85 0.16 75 / 0.18)', color: 'oklch(0.85 0.16 75)', borderRadius: 6, display: 'inline-flex', alignItems: 'center', justifyContent: 'center' }}><I.sparkle/></span>
                <div style={{ flex: 1, fontSize: 12.5 }}>
                  <span style={{ color: '#f4f4f0' }}>AI insight: </span>
                  <span style={{ color: '#a3a39c' }}>Drop-off terbesar di AI conversation → leads. Coba revisi prompt closing untuk naikin CVR ~5pt.</span>
                </div>
                <span style={{ fontSize: 11, color: ZZ_LIME }}>Apply suggestion <I.arrow/></span>
              </div>
            </div>

            {/* Workflows by revenue table: cols 1.6fr 1fr 1fr 1fr; each row has a lime underline bar (width p%) */}
            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 20 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
                <h3 style={{ fontSize: 16, fontWeight: 500, margin: 0 }}>Workflows by revenue</h3>
                <span className="mono" style={{ fontSize: 10.5, color: '#66665f' }}>30D</span>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1.6fr 1fr 1fr 1fr', fontSize: 10, color: '#66665f', padding: '0 0 8px', borderBottom: '1px solid #1a1a1d' }} className="mono tracked">
                <span>WORKFLOW</span><span style={{ textAlign: 'right' }}>RUNS</span><span style={{ textAlign: 'right' }}>CVR</span><span style={{ textAlign: 'right' }}>REVENUE</span>
              </div>
              {[
                ['promo-launch-mei',     12847, '24.2%', 'Rp 187jt', 1.0],
                ['giveaway-buku',         6312, '18.7%', 'Rp 89jt', 0.48],
                ['faq-bot-default',       8821, '12.4%', 'Rp 42jt', 0.22],
                ['lead-magnet-ebook',     3218, '21.8%', 'Rp 18jt', 0.10],
                ['quiz-style-guide',      1402,  '9.1%', 'Rp 8jt',  0.04],
                ['win-back-juni',          892,  '6.2%', 'Rp 4jt',  0.02],
              ].map(([n, r, c, rev, p], i) => (
                <div key={i} style={{ padding: '10px 0', borderBottom: '1px solid #1a1a1d', position: 'relative' }}>
                  <div style={{ display: 'grid', gridTemplateColumns: '1.6fr 1fr 1fr 1fr', fontSize: 12.5, alignItems: 'center', position: 'relative', zIndex: 1 }}>
                    <span className="mono">{n}</span>
                    <span className="mono tnum" style={{ textAlign: 'right' }}>{r.toLocaleString()}</span>
                    <span className="mono tnum" style={{ textAlign: 'right' }}>{c}</span>
                    <span className="mono tnum" style={{ textAlign: 'right', color: ZZ_LIME }}>{rev}</span>
                  </div>
                  <div style={{ position: 'absolute', left: 0, bottom: 0, height: 2, width: `${p * 100}%`, background: ZZ_LIME, opacity: 0.5 }}/>
                </div>
              ))}
            </div>
          </div>

          {/* Bottom widgets: 3 cols — Hourly heatmap, Top intents, Leaderboard akun */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12, marginTop: 12 }}>
            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 20 }}>
              <h3 style={{ fontSize: 14, fontWeight: 500, margin: '0 0 14px' }}>Hourly heatmap (comments)</h3>
              <Heatmap/>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 8, fontSize: 10, color: '#66665f' }} className="mono">
                <span>00</span><span>06</span><span>12</span><span>18</span><span>23</span>
              </div>
            </div>
            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 20 }}>
              <h3 style={{ fontSize: 14, fontWeight: 500, margin: '0 0 14px' }}>Top intents (AI)</h3>
              {[
                ['ask_price', 1842, ZZ_LIME],
                ['ask_availability', 1247, 'oklch(0.78 0.16 240)'],
                ['ask_size', 892, 'oklch(0.78 0.2 0)'],
                ['ask_shipping', 612, 'oklch(0.85 0.16 75)'],
                ['compliment', 287, '#66665f'],
                ['complaint', 142, '#66665f'],
              ].map(([k, n, c]) => (
                <div key={k} style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                  <span style={{ width: 6, height: 6, borderRadius: 999, background: c }}/>
                  <span className="mono" style={{ fontSize: 11.5, color: '#a3a39c', flex: 1 }}>{k}</span>
                  <span className="mono tnum" style={{ fontSize: 11.5 }}>{n}</span>
                </div>
              ))}
            </div>
            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 20 }}>
              <h3 style={{ fontSize: 14, fontWeight: 500, margin: '0 0 14px' }}>Leaderboard akun</h3>
              {[
                ['ataka.studio', 'Rp 187jt', 'AS', ZZ_LIME],
                ['folkstudio', 'Rp 89jt', 'F', 'oklch(0.78 0.2 0)'],
                ['rumah.kebun', 'Rp 42jt', 'RK', 'oklch(0.78 0.16 240)'],
                ['ekuator.co', 'Rp 18jt', 'E', 'oklch(0.85 0.16 75)'],
              ].map(([n, r, in_, c], i) => (
                <div key={n} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 0', borderTop: i ? '1px solid #1a1a1d' : 'none' }}>
                  <span className="mono" style={{ fontSize: 10.5, color: '#66665f', width: 14 }}>{i+1}</span>
                  <span style={{ width: 28, height: 28, borderRadius: 6, background: c, color: '#0a0a0a', display: 'inline-flex', alignItems: 'center', justifyContent: 'center', fontWeight: 600, fontSize: 11 }}>{in_}</span>
                  <span className="mono" style={{ fontSize: 12.5, flex: 1 }}>@{n}</span>
                  <span className="mono tnum" style={{ fontSize: 12, color: ZZ_LIME }}>{r}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function Big({ label, v, d, color }) {
  return (
    <div>
      <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f' }}>{label}</div>
      <div className="mono tnum" style={{ fontSize: 26, fontWeight: 500, marginTop: 4, letterSpacing: '-0.02em' }}>{v}</div>
      <div className="mono" style={{ fontSize: 11, color, marginTop: 2 }}>↑ {d}</div>
    </div>
  );
}

// BigChart: 3 area+line series over 28 days, max=200, grid lines at 25/50/75%, dashed "today" marker near right edge.
function BigChart() {
  const W = 1300, H = 220;
  const days = 28;
  const series = [
    { c: ZZ_LIME, base: 90, vary: 50, off: 0 },
    { c: 'oklch(0.78 0.16 240)', base: 60, vary: 40, off: 0.4 },
    { c: 'oklch(0.78 0.2 0)', base: 25, vary: 18, off: 0.9 },
  ];
  const make = (s) => Array.from({ length: days }, (_, i) => s.base + Math.sin((i + s.off * 7) * 0.6) * 12 + (i / days) * s.vary + Math.sin(i * 1.7) * 6);
  return (
    <svg width="100%" height={H} viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" style={{ display: 'block' }}>
      {[0.25, 0.5, 0.75].map(p => <line key={p} x1="0" x2={W} y1={H*p} y2={H*p} stroke="#1a1a1d"/>)}
      {series.map((s, i) => {
        const data = make(s);
        const max = 200;
        const pts = data.map((d, idx) => [idx * (W / (days - 1)), H - (d / max) * H]);
        const path = pts.map(([x, y], k) => (k ? 'L' : 'M') + x.toFixed(1) + ',' + y.toFixed(1)).join(' ');
        const area = path + ` L${W},${H} L0,${H} Z`;
        return (
          <g key={i}>
            <path d={area} fill={s.c} opacity={0.08}/>
            <path d={path} stroke={s.c} strokeWidth="1.6" fill="none"/>
          </g>
        );
      })}
      <line x1={W - 30} x2={W - 30} y1="0" y2={H} stroke={ZZ_LIME} strokeWidth="1" strokeDasharray="3 3" opacity="0.5"/>
      <circle cx={W - 30} cy={48} r="4" fill={ZZ_LIME}/>
    </svg>
  );
}

// Heatmap: 7 rows x 24 cols of lime cells (opacity by value). IMPORTANT for SSR: avoid Math.random()
// in render (hydration mismatch). Use a deterministic value function or precomputed fixture.
function Heatmap() {
  const days = 7, hours = 24;
  const cells = [];
  for (let d = 0; d < days; d++) {
    for (let h = 0; h < hours; h++) {
      const v = Math.max(0, Math.sin((h - 8) / 24 * Math.PI * 2) * 0.5 + 0.5 + Math.sin(d * 1.3) * 0.2 + (Math.random() * 0.2 - 0.1));
      cells.push({ d, h, v });
    }
  }
  return (
    <div style={{ display: 'grid', gridTemplateRows: `repeat(${days}, 1fr)`, gap: 2 }}>
      {Array.from({ length: days }).map((_, d) => (
        <div key={d} style={{ display: 'grid', gridTemplateColumns: `repeat(${hours}, 1fr)`, gap: 2 }}>
          {Array.from({ length: hours }).map((_, h) => {
            const c = cells[d * hours + h];
            return <span key={h} style={{ height: 12, background: `oklch(0.9 0.2 130 / ${(c.v * 0.85).toFixed(2)})`, borderRadius: 2 }}/>;
          })}
        </div>
      ))}
    </div>
  );
}
