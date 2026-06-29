// ─────────── ANALYTICS DRILLDOWN (single workflow) ─────────── (from menu-deeper-dark.jsx)
function AnalyticsDrilldownDark() {
  const W = 1440, H = 1100;
  return (
    <div style={{ width: W, height: H, background: '#0a0a0a', color: '#f4f4f0', fontFamily: 'var(--font-sans)', display: 'flex' }}>
      <Sidebar active="analytics"/>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {/* header: breadcrumb "Analytics / Workflow / launch-promo-mei" + Last 7d / Compare / Export ghost buttons */}
        <div style={{ height: 56, borderBottom: '1px solid #17171a', display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 24px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, fontSize: 13 }}>
            <span style={{ color: '#66665f' }}>Analytics</span>
            <span style={{ color: '#3a3a40' }}>/</span>
            <span style={{ color: '#a3a39c' }}>Workflow</span>
            <span style={{ color: '#3a3a40' }}>/</span>
            <span>launch-promo-mei</span>
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <button className="btn-ghost" style={{ padding: '7px 12px', fontSize: 12 }}>Last 7d ▾</button>
            <button className="btn-ghost" style={{ padding: '7px 12px', fontSize: 12 }}>Compare</button>
            <button className="btn-ghost" style={{ padding: '7px 12px', fontSize: 12 }}>Export</button>
          </div>
        </div>

        <div style={{ flex: 1, overflow: 'auto', padding: 24 }}>
          {/* hero metric row: 2fr (revenue card + line chart) + 1fr (step conversion bars) */}
          <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 14, marginBottom: 14 }}>
            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 22 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div>
                  <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f' }}>REVENUE INFLUENCED · 7D</div>
                  <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginTop: 6 }}>
                    <span className="mono" style={{ fontSize: 42, fontWeight: 500, letterSpacing: '-0.02em' }}>Rp 38.4jt</span>
                    <span className="mono" style={{ fontSize: 14, color: ZZ_LIME }}>+24.6%</span>
                  </div>
                  <div className="mono" style={{ fontSize: 11, color: '#66665f', marginTop: 4 }}>vs Rp 30.8jt · 7d prior</div>
                </div>
                <div style={{ display: 'flex', gap: 6 }}>
                  {['Revenue', 'Leads', 'Replies', 'Cost'].map((t, i) => (
                    <span key={t} className="mono" style={{ padding: '5px 10px', fontSize: 11, borderRadius: 6, background: i === 0 ? '#17171a' : 'transparent', color: i === 0 ? '#f4f4f0' : '#66665f', border: i === 0 ? '1px solid #232326' : 'none' }}>{t}</span>
                  ))}
                </div>
              </div>
              {/* big line chart with gradient fill, y-axis labels (0/10/20/30/40 jt), x days Mon..Sun, tooltip on Thu */}
              <svg viewBox="0 0 920 240" width="100%" height="240" style={{ marginTop: 18 }}>
                <defs>
                  <linearGradient id="rev-fill" x1="0" x2="0" y1="0" y2="1">
                    <stop offset="0%" stopColor={ZZ_LIME} stopOpacity="0.32"/>
                    <stop offset="100%" stopColor={ZZ_LIME} stopOpacity="0"/>
                  </linearGradient>
                </defs>
                {[0, 60, 120, 180].map(y => <line key={y} x1={40} x2={910} y1={y + 20} y2={y + 20} stroke="#1a1a1d"/>)}
                {[0, 10, 20, 30, 40].map((v, i) => (
                  <text key={v} x={32} y={200 - i * 45 + 4} fontFamily="var(--font-mono)" fontSize="10" fill="#66665f" textAnchor="end">{v}jt</text>
                ))}
                {(() => {
                  const days = ['Mon','Tue','Wed','Thu','Fri','Sat','Sun'];
                  const vals = [4.2, 5.8, 6.4, 8.1, 6.9, 4.1, 2.9];
                  const points = vals.map((v, i) => [60 + i * 130, 200 - (v / 40) * 180]);
                  const linePath = points.map((p, i) => `${i ? 'L' : 'M'}${p[0]} ${p[1]}`).join(' ');
                  const areaPath = `${linePath} L${points.at(-1)[0]} 200 L${points[0][0]} 200 Z`;
                  return (
                    <>
                      <path d={areaPath} fill="url(#rev-fill)"/>
                      <path d={linePath} fill="none" stroke={ZZ_LIME} strokeWidth="2"/>
                      {points.map((p, i) => (
                        <g key={i}>
                          <circle cx={p[0]} cy={p[1]} r="4" fill="#0a0a0a" stroke={ZZ_LIME} strokeWidth="2"/>
                          <text x={p[0]} y={222} fontFamily="var(--font-mono)" fontSize="10" fill="#66665f" textAnchor="middle">{days[i]}</text>
                          {i === 3 && (
                            <g>
                              <rect x={p[0] - 32} y={p[1] - 30} width={64} height={20} fill="#0a0a0a" stroke={ZZ_LIME} rx={3}/>
                              <text x={p[0]} y={p[1] - 16} fontFamily="var(--font-mono)" fontSize="11" fill={ZZ_LIME} textAnchor="middle">Rp 8.1jt</text>
                            </g>
                          )}
                        </g>
                      ))}
                    </>
                  );
                })()}
              </svg>
            </div>

            {/* step conversion bars: each step bar width=pct%, color fades via color-mix; drop pt label (pink) on right */}
            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 22 }}>
              <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f', marginBottom: 14 }}>STEP CONVERSION</div>
              {[
                ['Comment received', 4280, 100],
                ['Public reply sent', 4124, 96.4],
                ['DM delivered', 3812, 89.1],
                ['User replied', 1924, 44.9],
                ['Link clicked', 982, 22.9],
                ['Checkout started', 412, 9.6],
                ['Purchase', 218, 5.1],
              ].map(([k, n, pct], i, a) => (
                <div key={k} style={{ marginBottom: 10 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, marginBottom: 4 }}>
                    <span>{k}</span>
                    <span className="mono"><span>{n.toLocaleString()}</span><span style={{ color: '#66665f' }}> · {pct}%</span></span>
                  </div>
                  <div style={{ position: 'relative', height: 18, background: '#0a0a0a', borderRadius: 3, overflow: 'hidden' }}>
                    <div style={{ width: `${pct}%`, height: 18, background: i === 0 ? ZZ_LIME : `color-mix(in oklch, ${ZZ_LIME} ${100 - i * 12}%, #0a0a0a)`, borderRadius: 3 }}/>
                    {i > 0 && (
                      <span className="mono" style={{ position: 'absolute', right: 6, top: 2, fontSize: 9.5, color: 'oklch(0.78 0.2 0)' }}>−{(a[i-1][2] - pct).toFixed(1)}pt</span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* breakdown grids: 3 cols — By time of day (24 bars), By post (ranked list), By keyword intent (bars) */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 14 }}>
            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 18 }}>
              <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f', marginBottom: 12 }}>BY TIME OF DAY</div>
              {/* NOTE: avoid Math.random() in render for SSR — use deterministic values or a fixture */}
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(24, 1fr)', gap: 2, marginBottom: 8 }}>
                {Array.from({ length: 24 }, (_, h) => {
                  const v = Math.max(0.04, Math.sin((h - 4) * 0.3) * 0.5 + 0.4 + Math.random() * 0.2);
                  return <div key={h} style={{ height: 80, background: `color-mix(in oklch, ${ZZ_LIME} ${Math.round(v * 100)}%, #0a0a0a)`, borderRadius: 2 }}/>;
                })}
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }} className="mono">
                <span style={{ fontSize: 10, color: '#66665f' }}>00</span>
                <span style={{ fontSize: 10, color: '#66665f' }}>06</span>
                <span style={{ fontSize: 10, color: '#66665f' }}>12</span>
                <span style={{ fontSize: 10, color: '#66665f' }}>18</span>
                <span style={{ fontSize: 10, color: '#66665f' }}>23</span>
              </div>
              <div className="mono" style={{ fontSize: 11, color: '#a3a39c', marginTop: 8 }}>Peak <span style={{ color: ZZ_LIME }}>20:00 — 22:00</span> (32% of conv.)</div>
            </div>

            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 18 }}>
              <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f', marginBottom: 12 }}>BY POST</div>
              {[
                ['Bundle Mei 20% off', 1842, 'Rp 14.2jt'],
                ['Restock sage edition', 1124, 'Rp 9.8jt'],
                ['Behind the scenes', 718, 'Rp 6.4jt'],
                ['Limited drop juni', 412, 'Rp 4.1jt'],
                ['FAQ Q1', 184, 'Rp 3.9jt'],
              ].map(([t, n, r], i) => (
                <div key={t} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 0', borderTop: i ? '1px solid #1a1a1d' : 'none' }}>
                  <span className="mono" style={{ fontSize: 11, color: '#66665f', width: 18 }}>{i + 1}</span>
                  <div style={{ flex: 1 }}>
                    <div style={{ fontSize: 12.5 }}>{t}</div>
                    <div className="mono" style={{ fontSize: 10.5, color: '#66665f' }}>{n.toLocaleString()} comments</div>
                  </div>
                  <span className="mono" style={{ fontSize: 12, color: ZZ_LIME }}>{r}</span>
                </div>
              ))}
            </div>

            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 18 }}>
              <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f', marginBottom: 12 }}>BY KEYWORD INTENT</div>
              {[
                { l: 'ask_price', n: 1284, pct: 30 },
                { l: 'ask_link', n: 982, pct: 23 },
                { l: 'ask_size', n: 612, pct: 14 },
                { l: 'ask_shipping', n: 484, pct: 11 },
                { l: 'compliment', n: 412, pct: 9.6 },
                { l: 'other', n: 506, pct: 12.4 },
              ].map((r, i) => (
                <div key={r.l} style={{ marginBottom: 8 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, marginBottom: 3 }}>
                    <span className="mono" style={{ color: '#a3a39c' }}>{r.l}</span>
                    <span className="mono"><span style={{ color: '#f4f4f0' }}>{r.n}</span><span style={{ color: '#66665f' }}> · {r.pct}%</span></span>
                  </div>
                  <div style={{ height: 4, background: '#0a0a0a', borderRadius: 2 }}>
                    <div style={{ width: `${r.pct * 3}%`, height: 4, background: i === 0 ? ZZ_LIME : i === 1 ? 'oklch(0.78 0.16 240)' : 'oklch(0.85 0.16 75)', borderRadius: 2 }}/>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
