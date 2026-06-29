// ─────────── CONTACT PROFILE ─────────── (from menu-deeper-dark.jsx)
function ContactProfileDark() {
  const W = 1440, H = 1100;
  return (
    <div style={{ width: W, height: H, background: '#0a0a0a', color: '#f4f4f0', fontFamily: 'var(--font-sans)', display: 'flex' }}>
      <Sidebar active="contacts"/>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {/* header: breadcrumb "Contacts / Rina Susanti" + Add tag (ghost) + Send DM (ghost) + Add to flow (lime) */}
        <div style={{ height: 56, borderBottom: '1px solid #17171a', display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 24px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, fontSize: 13 }}>
            <span style={{ color: '#66665f' }}>Contacts</span>
            <span style={{ color: '#3a3a40' }}>/</span>
            <span>Rina Susanti</span>
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <button className="btn-ghost" style={{ padding: '7px 12px', fontSize: 12 }}>Add tag</button>
            <button className="btn-ghost" style={{ padding: '7px 12px', fontSize: 12 }}>Send DM</button>
            <button className="btn-lime" style={{ padding: '7px 12px', fontSize: 12 }}>Add to flow</button>
          </div>
        </div>

        <div style={{ flex: 1, overflow: 'auto' }}>
          {/* Hero: big avatar 84 + name + tag pills + meta line + 5 stat columns */}
          <div style={{ padding: '28px 32px', borderBottom: '1px solid #17171a', display: 'flex', alignItems: 'flex-start', gap: 20 }}>
            <Avatar name="RS" color="oklch(0.78 0.2 0)" size={84}/>
            <div style={{ flex: 1 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <h1 style={{ fontSize: 30, fontWeight: 500, letterSpacing: '-0.02em', margin: 0 }}>Rina Susanti</h1>
                <Pill tone="lime">warm-lead</Pill>
                <Pill tone="blue">jakarta</Pill>
                <Pill tone="neutral">first-time</Pill>
                <span style={{ fontSize: 12, color: '#66665f' }}>+ add tag</span>
              </div>
              <div className="mono" style={{ fontSize: 12.5, color: '#a3a39c', marginTop: 6 }}>@rina_susanti · 2,400 followers · joined zosmed list 28 Apr 2026</div>
              <div style={{ display: 'flex', gap: 24, marginTop: 18 }}>
                {[
                  ['LIFETIME VALUE', 'Rp 0', '#a3a39c'],
                  ['CONVERSATIONS', '12', '#a3a39c'],
                  ['LAST SEEN', '2 min ago', ZZ_LIME],
                  ['LEAD SCORE', '74 / 100', ZZ_LIME],
                  ['ATTRIBUTION', 'promo-launch-mei', '#a3a39c'],
                ].map(([k, v, c]) => (
                  <div key={k}>
                    <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f' }}>{k}</div>
                    <div className="mono" style={{ fontSize: 16, color: c, marginTop: 2 }}>{v}</div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Tabs (Activity active) — static tab strip, lime underline on active */}
          <div style={{ padding: '0 32px', borderBottom: '1px solid #17171a', display: 'flex', gap: 6 }}>
            {['Activity', 'Conversations', 'Properties', 'Notes', 'Lifecycle'].map((t, i) => (
              <span key={t} style={{
                padding: '12px 14px', fontSize: 13,
                color: i === 0 ? '#f4f4f0' : '#66665f',
                borderBottom: i === 0 ? `2px solid ${ZZ_LIME}` : '2px solid transparent',
                marginBottom: -1,
              }}>{t}</span>
            ))}
          </div>

          <div style={{ padding: 32, display: 'grid', gridTemplateColumns: '1.5fr 1fr', gap: 18 }}>
            {/* Activity timeline (left): vertical line + events with emoji icon ring, kind eyebrow, title, sub, optional quoted val box */}
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
                <h3 style={{ fontSize: 16, fontWeight: 500, margin: 0 }}>Activity timeline</h3>
                <span className="mono" style={{ fontSize: 10.5, color: '#66665f' }}>14 events · last 30 days</span>
              </div>
              <div style={{ position: 'relative' }}>
                <div style={{ position: 'absolute', left: 11, top: 6, bottom: 6, width: 1, background: '#232326' }}/>
                {[
                  { d: 'Today · 14:35', k: 'AI handled', i: '🤖', c: 'oklch(0.78 0.16 240)', t: 'AI replied with product link', sub: 'workflow: launch-promo-mei · matched intent: ask_product_link', val: '"Yang lagi promo ini kak: ataka.id/promo-mei…"' },
                  { d: 'Today · 14:32', k: 'DM received', i: '💬', c: ZZ_LIME, t: 'Sent DM via cek DM trigger', sub: 'delivered & read', val: '"Wah lumayan, ini link ke produk yang mana ya?"' },
                  { d: 'Today · 14:32', k: 'Public reply', i: '💬', c: ZZ_LIME, t: 'Auto-replied to comment', sub: 'on @ataka.studio post — "Bundle Mei 20% off"', val: '"Hai kak Rina! 👋 Cek DM ya, kita kirim detailnya 💚"' },
                  { d: 'Today · 14:32', k: 'Trigger', i: '⚡', c: 'oklch(0.85 0.16 75)', t: 'Triggered launch-promo-mei', sub: 'comment matched keyword "info"', val: '"info dong sis, harga berapa?"' },
                  { d: 'Today · 14:31', k: 'Tagged', i: '🏷', c: '#a3a39c', t: 'Auto-tagged: warm-lead', sub: 'rule: comment_intent_score > 0.6' },
                  { d: '28 Apr · 09:14', k: 'First touch', i: '✨', c: 'oklch(0.78 0.2 0)', t: 'Followed @ataka.studio', sub: 'first interaction with brand' },
                ].map((e, i) => (
                  <div key={i} style={{ position: 'relative', paddingLeft: 32, paddingBottom: 16 }}>
                    <span style={{ position: 'absolute', left: 0, top: 0, width: 22, height: 22, borderRadius: 999, background: '#111110', border: `1px solid ${e.c}`, display: 'inline-flex', alignItems: 'center', justifyContent: 'center', fontSize: 11 }}>{e.i}</span>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                      <span className="mono tracked" style={{ fontSize: 9.5, color: e.c }}>{e.k.toUpperCase()}</span>
                      <span className="mono" style={{ fontSize: 10.5, color: '#66665f' }}>{e.d}</span>
                    </div>
                    <div style={{ fontSize: 13, fontWeight: 500 }}>{e.t}</div>
                    <div style={{ fontSize: 12, color: '#a3a39c', marginTop: 2 }}>{e.sub}</div>
                    {e.val && <div className="mono" style={{ fontSize: 11.5, color: '#a3a39c', background: '#111110', border: '1px solid #1a1a1d', padding: '6px 10px', borderRadius: 6, marginTop: 6, lineHeight: 1.5 }}>{e.val}</div>}
                  </div>
                ))}
              </div>
            </div>

            {/* Right rail: Lead score breakdown (big 74/100 + sub-bars), Properties, In flows */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 18 }}>
                <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f', marginBottom: 12 }}>LEAD SCORE BREAKDOWN</div>
                <div style={{ display: 'flex', alignItems: 'baseline', gap: 6, marginBottom: 12 }}>
                  <span className="mono" style={{ fontSize: 36, fontWeight: 500 }}>74</span>
                  <span className="mono" style={{ fontSize: 14, color: '#66665f' }}>/ 100</span>
                  <span className="mono" style={{ marginLeft: 'auto', fontSize: 11, color: ZZ_LIME }}>+12 today</span>
                </div>
                {[
                  ['Engagement', 28, 30],
                  ['Intent signals', 24, 30],
                  ['Recency', 18, 20],
                  ['Reach', 4, 20],
                ].map(([k, v, max]) => (
                  <div key={k} style={{ marginBottom: 8 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11.5, marginBottom: 3 }}>
                      <span style={{ color: '#a3a39c' }}>{k}</span>
                      <span className="mono">{v} / {max}</span>
                    </div>
                    <div style={{ height: 4, background: '#17171a', borderRadius: 2 }}>
                      <div style={{ width: `${(v / max) * 100}%`, height: 4, background: ZZ_LIME, borderRadius: 2 }}/>
                    </div>
                  </div>
                ))}
              </div>

              <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 18 }}>
                <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f', marginBottom: 12 }}>PROPERTIES</div>
                {[
                  ['Email', '—'],
                  ['Phone', '—'],
                  ['City', 'Jakarta'],
                  ['Source', 'promo-launch-mei'],
                  ['First seen', '28 Apr 2026'],
                  ['IG verified', 'no'],
                  ['Language', 'id-ID'],
                ].map(([k, v]) => (
                  <div key={k} style={{ display: 'flex', justifyContent: 'space-between', padding: '6px 0', fontSize: 12, borderBottom: '1px solid #1a1a1d' }}>
                    <span style={{ color: '#66665f' }}>{k}</span>
                    <span className="mono" style={{ color: v === '—' ? '#3a3a40' : '#f4f4f0' }}>{v}</span>
                  </div>
                ))}
              </div>

              <div style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, padding: 18 }}>
                <div className="mono tracked" style={{ fontSize: 9.5, color: '#66665f', marginBottom: 12 }}>IN FLOWS · 2 ACTIVE</div>
                {[
                  ['launch-promo-mei', 'step 4 of 6 · DM sent', ZZ_LIME],
                  ['win-back-juni', 'queued · starts +28d', '#66665f'],
                ].map(([n, s, c]) => (
                  <div key={n} style={{ padding: '8px 0', borderBottom: '1px solid #1a1a1d' }}>
                    <div className="mono" style={{ fontSize: 12.5, color: '#f4f4f0' }}>{n}</div>
                    <div className="mono" style={{ fontSize: 10.5, color: c, marginTop: 2 }}>{s}</div>
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
