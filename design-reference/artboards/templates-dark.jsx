function TemplatesDark() {
  const W = 1440, H = 1100;
  const tpls = [
    { t: 'Launch Produk', d: 'Comment "info" → reply + DM checkout link + AI follow-up', tag: 'POPULAR', color: ZZ_LIME, runs: '2.4k', icon: <I.bolt/> },
    { t: 'Giveaway Entry', d: 'Auto-tag entry, validasi follow + share, draw winner', tag: 'POPULAR', color: 'oklch(0.78 0.2 0)', runs: '1.8k', icon: <I.heart/> },
    { t: 'Lead Magnet PDF', d: 'Komentar → DM kirim ebook/PDF + email capture', tag: '', color: 'oklch(0.78 0.16 240)', runs: '1.2k', icon: <I.send/> },
    { t: 'FAQ Bot Default', d: 'AI jawab harga, stok, ukuran, shipping pakai katalog', tag: 'AI', color: 'oklch(0.78 0.16 240)', runs: '912', icon: <I.ai/> },
    { t: 'Win-back 30d', d: 'Re-engage user yang DM 30 hari lalu tapi belum closing', tag: '', color: 'oklch(0.85 0.16 75)', runs: '742', icon: <I.user/> },
    { t: 'Quiz Funnel', d: 'Komentar trigger quiz → DM hasil + recommendation', tag: 'NEW', color: ZZ_LIME, runs: '512', icon: <I.sparkle/> },
    { t: 'Story Mention Welcome', d: 'User mention akun di Story → auto-reply DM thanks + intro', tag: '', color: '#3a3a40', runs: '418', icon: <I.sparkle/> },
    { t: 'Story Reply Funnel', d: 'Reply ke story tertentu → kirim link/CTA', tag: '', color: '#3a3a40', runs: '287', icon: <I.chat/> },
    { t: 'Customer Survey', d: 'Post-purchase survey via DM, collect rating', tag: '', color: '#3a3a40', runs: '142', icon: <I.check/> },
  ];
  return (
    <div style={{ width: W, height: H, background: '#0a0a0a', color: '#f4f4f0', fontFamily: 'var(--font-sans)', display: 'flex' }}>
      <Sidebar active="templates"/>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        <div style={{ height: 56, borderBottom: '1px solid #17171a', display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 24px' }}>
          <span style={{ fontSize: 14, fontWeight: 500 }}>Templates</span>
          <button className="btn-lime" style={{ padding: '7px 12px', fontSize: 12 }}><I.plus/> Submit template</button>
        </div>
        <div style={{ flex: 1, overflow: 'auto', padding: 24 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', marginBottom: 24 }}>
            <div>
              <span className="mono tracked" style={{ fontSize: 10, color: '#66665f' }}>54 TEMPLATES · CURATED BY ZOSMED + COMMUNITY</span>
              <h1 style={{ fontSize: 32, fontWeight: 500, letterSpacing: '-0.02em', margin: '6px 0 0' }}>Templates library</h1>
              <p style={{ fontSize: 14, color: '#a3a39c', margin: '6px 0 0', maxWidth: 540 }}>
                Workflow siap pakai. One-click clone ke workspace, edit sesuai brand, publish.
              </p>
            </div>
            <div style={{ display: 'flex', gap: 6 }}>
              {['All', 'Sales', 'Engagement', 'Lead-gen', 'AI', 'Support'].map((t, i) => (
                <span key={t} className="mono" style={{
                  padding: '6px 12px', fontSize: 11.5,
                  background: i === 0 ? ZZ_LIME : '#111110',
                  color: i === 0 ? '#0a0a0a' : '#a3a39c',
                  border: i === 0 ? 'none' : '1px solid #232326',
                  borderRadius: 999,
                }}>{t}</span>
              ))}
            </div>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 14 }}>
            {tpls.map((t, i) => (
              <div key={t.t} style={{ background: '#111110', border: '1px solid #232326', borderRadius: 12, overflow: 'hidden' }}>
                <div style={{ height: 120, background: `linear-gradient(135deg, color-mix(in oklch, ${t.color} 24%, transparent), #0a0a0a)`, position: 'relative', padding: 16, borderBottom: '1px solid #232326' }}>
                  <span style={{ width: 36, height: 36, background: '#0a0a0a', border: `1px solid ${t.color}`, color: t.color, borderRadius: 8, display: 'inline-flex', alignItems: 'center', justifyContent: 'center' }}>{t.icon}</span>
                  {t.tag && <span className="mono" style={{ position: 'absolute', top: 16, right: 16, padding: '3px 8px', fontSize: 10, background: t.color, color: '#0a0a0a', borderRadius: 4, fontWeight: 600 }}>{t.tag}</span>}
                  {/* mini node preview */}
                  <div style={{ display: 'flex', gap: 4, alignItems: 'center', marginTop: 18 }}>
                    {[0,1,2,3].map(k => (
                      <React.Fragment key={k}>
                        <span style={{ width: 22, height: 12, background: '#0a0a0a', border: `1px solid ${k === 0 ? t.color : '#2a2a2e'}`, borderRadius: 2 }}/>
                        {k < 3 && <span style={{ width: 8, height: 1, background: '#2a2a2e' }}/>}
                      </React.Fragment>
                    ))}
                  </div>
                </div>
                <div style={{ padding: 18 }}>
                  <h3 style={{ fontSize: 16, fontWeight: 500, margin: 0 }}>{t.t}</h3>
                  <p style={{ fontSize: 12.5, color: '#a3a39c', margin: '6px 0 12px', lineHeight: 1.5, minHeight: 36 }}>{t.d}</p>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingTop: 10, borderTop: '1px solid #17171a' }}>
                    <span className="mono" style={{ fontSize: 10.5, color: '#66665f' }}>{t.runs} clones · ★ 4.{4 + (i % 6)}</span>
                    <span style={{ fontSize: 12, color: ZZ_LIME, display: 'inline-flex', alignItems: 'center', gap: 4 }}>Use template <I.arrow/></span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

// ─────────── SETTINGS ───────────