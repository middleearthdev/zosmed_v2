function SafetyDark() {
  const W = 1440, H = 1100;
  return (
    <div style={{ width: W, height: H, background: '#0a0a0a', color: '#f4f4f0', fontFamily: 'var(--font-sans)', display: 'flex' }}>
      <Sidebar active="safety"/>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        <div style={{ height: 56, borderBottom: '1px solid #17171a', display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 24px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <span style={{ fontSize: 14, fontWeight: 500 }}>Safety center</span>
            <Pill tone="lime">● HEALTHY</Pill>
          </div>
          <span className="mono" style={{ fontSize: 11, color: '#66665f' }}>0 incidents · last 30 days</span>
        </div>
        <div style={{ flex: 1, overflow: 'auto', padding: 24 }}>
          <div style={{ marginBottom: 24 }}>
            <h1 style={{ fontSize: 32, fontWeight: 500, letterSpacing: '-0.02em', margin: 0 }}>Safety & rate limits</h1>
            <p style={{ fontSize: 14, color: '#a3a39c', margin: '6px 0 0', maxWidth: 640 }}>
              Pengaturan untuk jaga akun IG-mu aman dari deteksi spam. Default sudah konservatif — sesuaikan kalau kamu butuh throughput lebih.
            </p>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1.3fr 1fr', gap: 14, marginBottom: 14 }}>
            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 22 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                <h3 style={{ fontSize: 16, fontWeight: 500, margin: 0 }}>Rate limits</h3>
                <span className="mono" style={{ fontSize: 11, color: '#66665f' }}>per IG account</span>
              </div>
              {[
                { l: 'Comment replies / hour',  v: 142, cap: 750, rec: 'Meta private-reply ~750/hr' },
                { l: 'DMs sent / hour',         v: 89,  cap: 200, rec: 'safe ~200/hr · queue overflow' },
                { l: 'DMs sent / day',          v: 612, cap: 1000,rec: 'behaviour-based soft limit' },
                { l: 'Comments per post / 5min',v: 12,  cap: 30,  rec: 'human-paced' },
                { l: 'AI tokens / day',         v: 187000, cap: 1000000, rec: 'soft (cost guard)' },
              ].map(s => (
                <div key={s.l} style={{ marginBottom: 14 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, marginBottom: 6 }}>
                    <span>{s.l}</span>
                    <span className="mono"><span style={{ color: '#f4f4f0' }}>{s.v.toLocaleString()}</span><span style={{ color: '#66665f' }}> / {s.cap.toLocaleString()}</span></span>
                  </div>
                  <div style={{ position: 'relative', height: 6, background: '#17171a', borderRadius: 3 }}>
                    <div style={{ width: `${Math.min(100, (s.v / s.cap) * 100)}%`, height: 6, background: ZZ_LIME, borderRadius: 3 }}/>
                    <div style={{ position: 'absolute', left: '80%', top: -2, width: 1, height: 10, background: 'oklch(0.85 0.16 75)' }}/>
                  </div>
                  <div className="mono" style={{ fontSize: 10.5, color: '#66665f', marginTop: 4 }}>{s.rec}</div>
                </div>
              ))}
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 22 }}>
                <h3 style={{ fontSize: 16, fontWeight: 500, margin: '0 0 12px' }}>Anti-spam patterns</h3>
                {[
                  ['Random delay 2–8s before reply', true, 'human-paced response'],
                  ['Skip duplicate DM to same user', true, 'no spam-burst'],
                  ['Pause if error rate > 5%', true, 'auto-circuit-breaker'],
                  ['Skip reply if comment < 3 chars', false, 'avoid emoji-only spam'],
                  ['Mirror user language (ID/EN)', true, 'natural conversation'],
                  ['Quiet hours (22:00 — 06:00)', false, 'optional'],
                ].map(([l, on, sub]) => (
                  <div key={l} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '10px 0', borderTop: '1px solid #1a1a1d' }}>
                    <span style={{ width: 32, height: 18, borderRadius: 999, background: on ? ZZ_LIME : '#2a2a2e', position: 'relative', flexShrink: 0 }}>
                      <span style={{ position: 'absolute', top: 2, left: on ? 16 : 2, width: 14, height: 14, background: '#0a0a0a', borderRadius: 999, transition: 'left .2s' }}/>
                    </span>
                    <div style={{ flex: 1 }}>
                      <div style={{ fontSize: 13 }}>{l}</div>
                      <div className="mono" style={{ fontSize: 10.5, color: '#66665f' }}>{sub}</div>
                    </div>
                  </div>
                ))}
              </div>

              <div style={{ background: '#111110', border: '1px solid oklch(0.85 0.16 75 / 0.4)', borderRadius: 12, padding: 22 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10 }}>
                  <span style={{ width: 28, height: 28, background: 'oklch(0.85 0.16 75 / 0.18)', color: 'oklch(0.85 0.16 75)', borderRadius: 6, display: 'inline-flex', alignItems: 'center', justifyContent: 'center' }}><I.shield/></span>
                  <h3 style={{ fontSize: 14, fontWeight: 500, margin: 0 }}>Kill switch</h3>
                </div>
                <p style={{ fontSize: 12.5, color: '#a3a39c', margin: '0 0 12px', lineHeight: 1.5 }}>
                  Pause semua workflow untuk akun ini secara instan. Berguna kalau kamu lihat anomaly atau dapet warning dari IG.
                </p>
                <button className="btn-ghost" style={{ width: '100%', justifyContent: 'center', borderColor: 'oklch(0.85 0.16 75)', color: 'oklch(0.85 0.16 75)' }}>
                  ⏸ Pause all workflows
                </button>
              </div>
            </div>
          </div>

          <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 22 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
              <h3 style={{ fontSize: 16, fontWeight: 500, margin: 0 }}>Activity log</h3>
              <span className="mono" style={{ fontSize: 11, color: '#66665f' }}>last 24 hours · 8 events</span>
            </div>
            {[
              ['14:32', 'auto-pause', 'rate near limit (89/100 dm/hr)', 'system', 'oklch(0.85 0.16 75)'],
              ['14:38', 'resumed',    'cooldown done — back to normal', 'system', ZZ_LIME],
              ['12:18', 'rule-skip',  'duplicate DM blocked — @rina_susanti', 'anti-spam', ZZ_LIME],
              ['09:42', 'config',     'rate limit set: 150 → 200 dm/hr', 'maya', '#a3a39c'],
              ['09:01', 'workflow',   'win-back-juni paused by user', 'maya', '#a3a39c'],
              ['Yesterday 23:50', 'health-check', 'all systems normal · IG token valid', 'system', '#a3a39c'],
            ].map(([t, k, msg, who, color], i) => (
              <div key={i} style={{ display: 'grid', gridTemplateColumns: '120px 100px 1fr 100px', padding: '10px 0', borderTop: i ? '1px solid #1a1a1d' : 'none', alignItems: 'center', gap: 12 }}>
                <span className="mono" style={{ fontSize: 11, color: '#66665f' }}>{t}</span>
                <span className="mono" style={{ fontSize: 11, color, padding: '2px 8px', background: `color-mix(in oklch, ${color} 14%, transparent)`, borderRadius: 3, display: 'inline-block', justifySelf: 'flex-start' }}>{k}</span>
                <span style={{ fontSize: 13, color: '#a3a39c' }}>{msg}</span>
                <span className="mono" style={{ fontSize: 11, color: '#66665f', textAlign: 'right' }}>by {who}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}