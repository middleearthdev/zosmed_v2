function SettingsDark() {
  const W = 1440, H = 1100;
  return (
    <div style={{ width: W, height: H, background: '#0a0a0a', color: '#f4f4f0', fontFamily: 'var(--font-sans)', display: 'flex' }}>
      <Sidebar active="settings"/>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        <div style={{ height: 56, borderBottom: '1px solid #17171a', display: 'flex', alignItems: 'center', padding: '0 24px' }}>
          <span style={{ fontSize: 14, fontWeight: 500 }}>Settings</span>
        </div>
        <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
          <div style={{ width: 220, borderRight: '1px solid #17171a', padding: 18 }}>
            {['Workspace', 'Members', 'Connected accounts', 'Billing', 'Plans', 'API & Webhooks', 'Notifications', 'Security', 'Danger zone'].map((t, i) => (
              <div key={t} style={{
                padding: '8px 12px', fontSize: 13, borderRadius: 6,
                background: i === 2 ? '#17171a' : 'transparent',
                color: i === 2 ? '#f4f4f0' : '#a3a39c',
                marginBottom: 2,
                borderLeft: i === 2 ? `2px solid ${ZZ_LIME}` : '2px solid transparent',
                paddingLeft: 10,
              }}>{t}</div>
            ))}
          </div>
          <div style={{ flex: 1, overflow: 'auto', padding: 32 }}>
            <h1 style={{ fontSize: 32, fontWeight: 500, letterSpacing: '-0.02em', margin: '0 0 6px' }}>Connected accounts</h1>
            <p style={{ fontSize: 14, color: '#a3a39c', margin: '0 0 28px', maxWidth: 600 }}>
              Hubungkan akun Instagram untuk diatur otomatis. Workspace Pro support hingga 5 akun · Enterprise unlimited.
            </p>

            {/* Connected accounts */}
            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 4, marginBottom: 14 }}>
              {[
                { ig: '@ataka.studio', n: 'Ataka Studio', f: '24,800', primary: true, perm: 'full', last: 'just now' },
                { ig: '@ataka.archive', n: 'Ataka Archive', f: '8,200', perm: 'read-only', last: '2h ago' },
                { ig: '@maya.personal', n: 'Maya Rahma', f: '1,400', perm: 'full', last: '1d ago' },
              ].map((a, i) => (
                <div key={a.ig} style={{ display: 'grid', gridTemplateColumns: '40px 1.4fr 0.8fr 0.8fr 1fr 80px', padding: '14px 16px', alignItems: 'center', gap: 12, borderBottom: i < 2 ? '1px solid #17171a' : 'none' }}>
                  <Avatar name={a.ig.slice(1, 3).toUpperCase()} color={i === 0 ? ZZ_LIME : '#3a3a40'} size={36}/>
                  <div>
                    <div style={{ fontSize: 14, fontWeight: 500, display: 'flex', alignItems: 'center', gap: 6 }}>
                      {a.n}
                      {a.primary && <Pill tone="lime">PRIMARY</Pill>}
                    </div>
                    <div className="mono" style={{ fontSize: 11.5, color: '#66665f' }}>{a.ig} · {a.f} followers</div>
                  </div>
                  <Pill tone={a.perm === 'full' ? 'lime' : 'neutral'}>{a.perm}</Pill>
                  <span className="mono" style={{ fontSize: 11, color: '#66665f' }}>last sync {a.last}</span>
                  <div style={{ display: 'flex', gap: 4, fontSize: 11.5 }}>
                    <span style={{ color: '#a3a39c' }}>Configure</span>
                    <span style={{ color: '#3a3a40' }}>·</span>
                    <span style={{ color: '#a3a39c' }}>Re-auth</span>
                  </div>
                  <span className="mono" style={{ fontSize: 11, color: 'oklch(0.78 0.2 0)', textAlign: 'right' }}>Disconnect</span>
                </div>
              ))}
              <div style={{ padding: 14, borderTop: '1px solid #17171a', display: 'flex', alignItems: 'center', gap: 10 }}>
                <span style={{ width: 36, height: 36, border: '1px dashed #2a2a2e', borderRadius: 8, display: 'inline-flex', alignItems: 'center', justifyContent: 'center', color: '#66665f' }}><I.plus/></span>
                <span style={{ fontSize: 13, color: '#a3a39c' }}>Connect another Instagram account</span>
                <button className="btn-ghost" style={{ marginLeft: 'auto', padding: '6px 12px', fontSize: 12 }}>Connect</button>
              </div>
            </div>

            <h2 style={{ fontSize: 18, fontWeight: 500, margin: '32px 0 12px' }}>Other integrations</h2>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12 }}>
              {[
                { n: 'Google Sheets', s: 'connected', d: 'Sync leads to your sheet', c: ZZ_LIME },
                { n: 'Notion', s: 'connected', d: 'Mirror contacts as DB', c: ZZ_LIME },
                { n: 'Meta Pixel', s: 'connected', d: 'Track conversion events', c: ZZ_LIME },
                { n: 'Webhook', s: 'configured', d: '2 endpoints active', c: 'oklch(0.78 0.16 240)' },
                { n: 'Slack', s: 'available', d: 'Notify lead alerts to channel', c: '#3a3a40' },
                { n: 'WhatsApp Business', s: 'soon', d: 'Cross-channel funnel', c: '#3a3a40' },
              ].map(it => (
                <div key={it.n} style={{ background: '#111110', border: '1px solid #232326', borderRadius: 10, padding: 16, display: 'flex', flexDirection: 'column', gap: 8 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span style={{ width: 32, height: 32, borderRadius: 6, background: '#17171a', color: it.c, display: 'inline-flex', alignItems: 'center', justifyContent: 'center', fontFamily: 'var(--font-mono)', fontSize: 14 }}>⌬</span>
                    <Pill tone={it.s === 'connected' || it.s === 'configured' ? 'lime' : 'neutral'}>{it.s}</Pill>
                  </div>
                  <div style={{ fontSize: 14, fontWeight: 500 }}>{it.n}</div>
                  <div style={{ fontSize: 12, color: '#a3a39c' }}>{it.d}</div>
                </div>
              ))}
            </div>

            <h2 style={{ fontSize: 18, fontWeight: 500, margin: '32px 0 12px' }}>Plan & usage</h2>
            <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 22, display: 'grid', gridTemplateColumns: '1.4fr 1fr', gap: 24, alignItems: 'center' }}>
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <span style={{ fontSize: 22, fontWeight: 500 }}>Pro plan</span>
                  <Pill tone="lime">CURRENT</Pill>
                </div>
                <div className="mono" style={{ fontSize: 13, color: '#a3a39c', marginTop: 6 }}>Rp 490,000 / bulan · billed monthly</div>
                <div style={{ display: 'flex', gap: 16, marginTop: 16 }}>
                  <div>
                    <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f' }}>NEXT INVOICE</div>
                    <div className="mono" style={{ fontSize: 14, marginTop: 2 }}>15 May 2026</div>
                  </div>
                  <div>
                    <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f' }}>PAYMENT</div>
                    <div className="mono" style={{ fontSize: 14, marginTop: 2 }}>•••• 4821</div>
                  </div>
                </div>
              </div>
              <div>
                {[
                  ['Auto-DMs', 892, 2500],
                  ['AI tokens', 187000, 1000000],
                  ['IG accounts', 3, 5],
                  ['Workflows', 6, 20],
                ].map(([k, v, cap]) => (
                  <div key={k} style={{ marginBottom: 10 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, marginBottom: 4 }}>
                      <span style={{ color: '#a3a39c' }}>{k}</span>
                      <span className="mono"><span>{v.toLocaleString()}</span><span style={{ color: '#66665f' }}> / {cap.toLocaleString()}</span></span>
                    </div>
                    <div style={{ height: 4, background: '#17171a', borderRadius: 2 }}>
                      <div style={{ width: `${(v / cap) * 100}%`, height: 4, background: ZZ_LIME, borderRadius: 2 }}/>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}