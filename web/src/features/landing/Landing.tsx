import type { MouseEvent } from 'react'
import './landing.css'

const GITHUB_URL = 'https://github.com/'

// The landing page is the app's front door. "Explore the live demo" navigates
// internally to /app (which lands on add-repo when empty, or the workspace when
// repositories already exist); onEnterDemo performs that client-side navigation.
export function Landing({ onEnterDemo }: { onEnterDemo: () => void }) {
  const enterDemo = (event: MouseEvent<HTMLAnchorElement>) => {
    // Let modified clicks (new tab, etc.) fall through to the real /app href.
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
    event.preventDefault()
    onEnterDemo()
  }

  return (
    <div className="landing">
      <header className="bar">
        <div className="wrap bar-in">
          <div className="brand">
            <svg className="mark" viewBox="0 0 32 32" fill="none" aria-hidden="true">
              <circle cx="16" cy="16" r="4.4" fill="#2fd6bb" />
              <circle
                cx="16"
                cy="16"
                r="9"
                stroke="#2fd6bb"
                strokeOpacity="0.5"
                strokeWidth="1.4"
              />
              <circle
                cx="16"
                cy="16"
                r="13.5"
                stroke="#2fd6bb"
                strokeOpacity="0.22"
                strokeWidth="1.4"
              />
              <circle cx="27" cy="9" r="2.4" fill="#f36a5d" />
              <circle cx="6" cy="12" r="2.4" fill="#56a6f0" />
              <circle cx="24" cy="26" r="2.4" fill="#f36a5d" />
            </svg>
            CodeAtlas
          </div>
          <nav className="bar-links">
            <a className="navlink" href="#question">
              Blast radius
            </a>
            <a className="navlink" href="#why">
              Why it&apos;s different
            </a>
            <a className="navlink" href="#how">
              How it works
            </a>
            <a
              className="btn btn-ghost navlink"
              href={GITHUB_URL}
              target="_blank"
              rel="noopener noreferrer"
            >
              GitHub ↗
            </a>
            <a className="btn btn-primary" href="/app" onClick={enterDemo}>
              Explore the live demo
            </a>
          </nav>
        </div>
      </header>

      <main>
        {/* ============ HERO ============ */}
        <section className="wrap hero">
          <div className="hero-copy">
            <span className="hero-eyebrow eyebrow">
              <span className="dot" />
              Code intelligence · local-first · no AI
            </span>
            <h1 className="headline">
              See what <span className="em">breaks</span> before you touch it.
            </h1>
            <p className="subhead">
              Click any function, file, or module and CodeAtlas lights up its blast radius across
              the whole codebase — everything that depends on it, and everything it relies on. One{' '}
              <code>codeatlas serve</code>, no database to run, nothing leaves your machine.
            </p>
            <div className="cta-row">
              <a
                className="btn btn-primary"
                href="/app"
                onClick={enterDemo}
                aria-label="Explore the live demo"
              >
                Explore the live demo
                <svg width="15" height="15" viewBox="0 0 16 16" fill="none">
                  <path
                    d="M4 8h8M8 4l4 4-4 4"
                    stroke="currentColor"
                    strokeWidth="1.6"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
              </a>
              <a
                className="btn btn-ghost"
                href={GITHUB_URL}
                target="_blank"
                rel="noopener noreferrer"
              >
                View source
              </a>
            </div>
            <div className="cta-note">
              <svg width="14" height="14" viewBox="0 0 16 16" fill="none">
                <path
                  d="M8 1.5l5.5 2.4v3.7c0 3.4-2.3 5.7-5.5 6.9-3.2-1.2-5.5-3.5-5.5-6.9V3.9L8 1.5z"
                  stroke="#2fd6bb"
                  strokeWidth="1.3"
                  strokeLinejoin="round"
                />
              </svg>
              Analyzes locally · source never uploaded · deterministic
            </div>
          </div>

          {/* live blast-radius graph */}
          <div
            className="stage"
            aria-label="A dependency graph showing a selected function and its blast radius"
          >
            <div className="stage-head">
              <span className="tl">
                <i />
                <i />
                <i />
              </span>
              impact-map · orders.ts::createOrder
            </div>
            <svg
              viewBox="0 0 520 430"
              role="img"
              aria-label="createOrder at the center, with three dependents that break and two dependencies it relies on"
            >
              <defs>
                <marker id="ah" markerWidth="7" markerHeight="7" refX="5.4" refY="3" orient="auto">
                  <path d="M0 0l6 3-6 3z" fill="var(--l-line-strong)" />
                </marker>
              </defs>

              {/* edges: dependents (rose, they point INTO the epicenter) */}
              <path
                className="edge"
                d="M175 78 C 210 130, 235 160, 258 196"
                stroke="var(--l-rose)"
                strokeOpacity="0.55"
              />
              <path
                className="edge"
                d="M418 96 C 370 140, 330 170, 290 200"
                stroke="var(--l-rose)"
                strokeOpacity="0.55"
              />
              <path
                className="edge"
                d="M436 232 C 390 226, 340 220, 300 216"
                stroke="var(--l-rose)"
                strokeOpacity="0.55"
              />
              {/* edges: dependencies (sky, epicenter points OUT to them) */}
              <path
                className="edge"
                d="M232 224 C 190 250, 150 268, 118 286"
                stroke="var(--l-sky)"
                strokeOpacity="0.5"
                markerEnd="url(#ah)"
              />
              <path
                className="edge"
                d="M262 244 C 258 300, 268 340, 300 372"
                stroke="var(--l-sky)"
                strokeOpacity="0.5"
                markerEnd="url(#ah)"
              />

              {/* ripple rings from epicenter */}
              <circle className="ripple r1" cx="260" cy="212" r="118" strokeWidth="1.4" />
              <circle className="ripple r2" cx="260" cy="212" r="118" strokeWidth="1.4" />
              <circle className="ripple r3" cx="260" cy="212" r="118" strokeWidth="1.4" />

              {/* dependent nodes (rose) */}
              <g className="node">
                <rect
                  x="112"
                  y="52"
                  width="118"
                  height="42"
                  rx="9"
                  fill="#1a222b"
                  stroke="var(--l-rose)"
                  strokeOpacity="0.7"
                />
                <text className="node-kind" x="124" y="70" fill="var(--l-rose)">
                  function · dependent
                </text>
                <text className="node-label" x="124" y="84" fill="#eef2f4">
                  checkout()
                </text>
              </g>
              <g className="node">
                <rect
                  x="392"
                  y="72"
                  width="112"
                  height="42"
                  rx="9"
                  fill="#1a222b"
                  stroke="var(--l-rose)"
                  strokeOpacity="0.7"
                />
                <text className="node-kind" x="404" y="90" fill="var(--l-rose)">
                  route · dependent
                </text>
                <text className="node-label" x="404" y="104" fill="#eef2f4">
                  POST /cart
                </text>
              </g>
              <g className="node">
                <rect
                  x="398"
                  y="210"
                  width="118"
                  height="42"
                  rx="9"
                  fill="#1a222b"
                  stroke="var(--l-rose)"
                  strokeOpacity="0.7"
                />
                <text className="node-kind" x="410" y="228" fill="var(--l-rose)">
                  component · dep.
                </text>
                <text className="node-label" x="410" y="242" fill="#eef2f4">
                  OrderList.tsx
                </text>
              </g>

              {/* dependency nodes (sky) */}
              <g className="node">
                <rect
                  x="36"
                  y="266"
                  width="108"
                  height="42"
                  rx="9"
                  fill="#1a222b"
                  stroke="var(--l-sky)"
                  strokeOpacity="0.7"
                />
                <text className="node-kind" x="48" y="284" fill="var(--l-sky)">
                  module · relied-on
                </text>
                <text className="node-label" x="48" y="298" fill="#eef2f4">
                  db/query
                </text>
              </g>
              <g className="node">
                <rect
                  x="252"
                  y="372"
                  width="112"
                  height="42"
                  rx="9"
                  fill="#1a222b"
                  stroke="var(--l-sky)"
                  strokeOpacity="0.7"
                />
                <text className="node-kind" x="264" y="390" fill="var(--l-sky)">
                  util · relied-on
                </text>
                <text className="node-label" x="264" y="404" fill="#eef2f4">
                  money.ts
                </text>
              </g>

              {/* epicenter */}
              <g className="epi">
                <circle
                  className="epi-core"
                  cx="260"
                  cy="212"
                  r="30"
                  fill="rgba(47,214,187,0.12)"
                  stroke="var(--l-teal)"
                  strokeWidth="1.6"
                />
                <rect
                  x="196"
                  y="196"
                  width="128"
                  height="46"
                  rx="10"
                  fill="#0e2a25"
                  stroke="var(--l-teal)"
                  strokeWidth="1.7"
                />
                <text className="node-kind" x="210" y="215" fill="var(--l-teal)">
                  function · epicenter
                </text>
                <text
                  className="node-label"
                  x="210"
                  y="230"
                  fill="#eef2f4"
                  style={{ fontWeight: 600 }}
                >
                  createOrder()
                </text>
              </g>
            </svg>
          </div>
        </section>

        <div className="wrap">
          <div className="divider" />
        </div>

        {/* ============ THE QUESTION ============ */}
        <section className="wrap" id="question">
          <div className="sec-head">
            <span className="eyebrow">The one question</span>
            <h2>&ldquo;If I change this, what does it affect?&rdquo;</h2>
            <p>
              Every node you select becomes an epicenter. CodeAtlas walks the dependency graph in
              both directions and paints the result by role — so the impact reads at a glance,
              before you write a line.
            </p>
          </div>
          <div className="legend">
            <div className="leg teal">
              <span className="swatch">
                <b />
                Epicenter
              </span>
              <h3>What you selected</h3>
              <p>The file, module, or function at the center of the change.</p>
            </div>
            <div className="leg rose">
              <span className="swatch">
                <b />
                Breaks
              </span>
              <h3>Dependents</h3>
              <p>Everything that calls or imports it — the code that could break.</p>
            </div>
            <div className="leg sky">
              <span className="swatch">
                <b />
                Relies on
              </span>
              <h3>Dependencies</h3>
              <p>What the epicenter itself leans on to do its job.</p>
            </div>
            <div className="leg amber">
              <span className="swatch">
                <b />
                Changed
              </span>
              <h3>Review mode</h3>
              <p>Seed the radius from your uncommitted edits vs. HEAD.</p>
            </div>
          </div>
        </section>

        {/* ============ WHY DIFFERENT ============ */}
        <section className="wrap" id="why">
          <div className="sec-head">
            <span className="eyebrow">Why it&apos;s different</span>
            <h2>Built to be trusted, not to phone home.</h2>
            <p>
              Three deliberate constraints that make CodeAtlas something you can point at a private
              repo without a second thought.
            </p>
          </div>
          <div className="pillars">
            <div className="pillar">
              <div className="ic">
                <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                  <path
                    d="M10 2l6.5 2.9v4.4c0 4-2.7 6.8-6.5 8.2-3.8-1.4-6.5-4.2-6.5-8.2V4.9L10 2z"
                    stroke="#2fd6bb"
                    strokeWidth="1.4"
                    strokeLinejoin="round"
                  />
                  <path
                    d="M7 10l2 2 4-4.2"
                    stroke="#2fd6bb"
                    strokeWidth="1.4"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
              </div>
              <h3>Local-first &amp; private</h3>
              <p>
                Your source is parsed on your machine and never uploaded. CodeAtlas stores only
                derived graph metadata and file hashes — never the code itself.
              </p>
              <span className="tag">source stays home</span>
            </div>
            <div className="pillar">
              <div className="ic">
                <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                  <rect
                    x="3"
                    y="3"
                    width="14"
                    height="14"
                    rx="3"
                    stroke="#56a6f0"
                    strokeWidth="1.4"
                  />
                  <path
                    d="M7 8l2 2-2 2M11 12h2"
                    stroke="#56a6f0"
                    strokeWidth="1.4"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
              </div>
              <h3>One binary, zero setup</h3>
              <p>
                Embedded SQLite means no Postgres, no Docker, no services to run. Download,{' '}
                <code className="mono-sky">serve</code>, open the browser. The whole database is one
                file.
              </p>
              <span className="tag">no external database</span>
            </div>
            <div className="pillar">
              <div className="ic">
                <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                  <circle cx="10" cy="10" r="7" stroke="#f4b13c" strokeWidth="1.4" />
                  <path
                    d="M10 6v4l2.6 1.6"
                    stroke="#f4b13c"
                    strokeWidth="1.4"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
              </div>
              <h3>No AI. On purpose.</h3>
              <p>
                Every edge comes from parsing real syntax with Tree-sitter — not a model&apos;s
                guess. Same repo, same graph, every time. Determinism is the feature.
              </p>
              <span className="tag">reproducible by design</span>
            </div>
          </div>
        </section>

        <div className="wrap">
          <div className="divider" />
        </div>

        {/* ============ HOW IT WORKS ============ */}
        <section className="wrap" id="how">
          <div className="sec-head">
            <span className="eyebrow">Under the hood</span>
            <h2>From source files to a blast radius.</h2>
            <p>
              Four stages turn a folder of code into an interactive dependency map — streamed live
              as it runs.
            </p>
          </div>
          <div className="pipe">
            <div className="steps">
              <div className="step">
                <span className="num">01</span>
                <div>
                  <h3>Parse with Tree-sitter</h3>
                  <p>
                    Each file is parsed into a concrete syntax tree; functions, files, and modules
                    become <code>entities</code>. A pure-Go fallback keeps it building even without
                    a C toolchain.
                  </p>
                </div>
              </div>
              <div className="step">
                <span className="num">02</span>
                <div>
                  <h3>Resolve relationships</h3>
                  <p>
                    Imports, calls, and references are resolved into typed edges between entities —
                    the raw graph, with a confidence score on every link.
                  </p>
                </div>
              </div>
              <div className="step">
                <span className="num">03</span>
                <div>
                  <h3>Traverse both directions</h3>
                  <p>
                    A recursive SQL walk fans out from the seed — upstream to dependents, downstream
                    to dependencies — with a cycle guard and a depth limit.
                  </p>
                </div>
              </div>
              <div className="step">
                <span className="num">04</span>
                <div>
                  <h3>Render the map</h3>
                  <p>
                    React Flow + an ELK layout draw the graph in the browser; the selected
                    node&apos;s radius is merged on as a color-coded overlay, rolled up to file
                    level.
                  </p>
                </div>
              </div>
            </div>

            <div className="codecard">
              <div className="ch">
                <i />
                store/graph.go — impactFrom()
              </div>
              <pre>
                <span className="cm">-- fan out from the seed entity, both ways</span>
                {'\n'}
                <span className="kw">WITH RECURSIVE</span> <span className="fn">radius</span>(id,
                depth, dir) <span className="kw">AS</span> ({'\n'}{' '}
                <span className="kw">SELECT</span> id, <span className="st">0</span>,{' '}
                <span className="st">&apos;root&apos;</span> <span className="kw">FROM</span>{' '}
                entities{'\n'} <span className="kw">WHERE</span> id <span className="kw">IN</span> (
                <span className="cm">/* seeds */</span>){'\n'} <span className="kw">UNION</span>
                {'\n'} <span className="kw">SELECT</span> e.target, r.depth{' '}
                <span className="st">+ 1</span>,{'\n'}{' '}
                <span className="st">&apos;dependent&apos;</span>
                {'\n'} <span className="kw">FROM</span> relationships e{'\n'}{' '}
                <span className="kw">JOIN</span> <span className="fn">radius</span> r{' '}
                <span className="kw">ON</span> e.source = r.id{'\n'}{' '}
                <span className="kw">WHERE</span> r.depth &lt; <span className="st">:maxDepth</span>
                {'\n'}){'\n'}
                <span className="kw">SELECT</span> <span className="kw">DISTINCT</span> id, dir{' '}
                <span className="kw">FROM</span> <span className="fn">radius</span>;
              </pre>
            </div>
          </div>
        </section>

        {/* ============ REVIEW MODE ============ */}
        <section className="wrap">
          <div className="review">
            <div>
              <span className="eyebrow">Review changes</span>
              <h3>Turn your working tree into a blast radius.</h3>
              <p>
                CodeAtlas diffs your uncommitted edits against{' '}
                <code className="mono-amber">HEAD</code>, maps the changed line ranges back to
                entities, and runs the same impact walk over them — so you see the reach of a change
                while you&apos;re still writing it.
              </p>
            </div>
            <div className="rstats">
              <div className="rstat a">
                <div className="n">3</div>
                <div className="l">files changed</div>
              </div>
              <div className="rstat a">
                <div className="n">7</div>
                <div className="l">symbols changed</div>
              </div>
              <div className="rstat b">
                <div className="n">18</div>
                <div className="l">dependents hit</div>
              </div>
              <div className="rstat b">
                <div className="n">2</div>
                <div className="l">routes affected</div>
              </div>
            </div>
          </div>
        </section>

        {/* ============ FINAL CTA ============ */}
        <section className="wrap">
          <div className="final">
            <span className="eyebrow">Open source · MIT</span>
            <h2>Map your codebase in one command.</h2>
            <p>
              Point CodeAtlas at a local folder or a Git URL, watch it stream the parse, then start
              clicking. No account, no upload, no setup.
            </p>
            <div className="cta-row">
              <a className="btn btn-primary" href="/app" onClick={enterDemo}>
                Explore the live demo
                <svg width="15" height="15" viewBox="0 0 16 16" fill="none">
                  <path
                    d="M4 8h8M8 4l4 4-4 4"
                    stroke="currentColor"
                    strokeWidth="1.6"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
              </a>
              <a
                className="btn btn-ghost"
                href={GITHUB_URL}
                target="_blank"
                rel="noopener noreferrer"
              >
                Read the docs
              </a>
            </div>
          </div>
        </section>
      </main>

      <footer>
        <div className="wrap foot">
          <div className="stack">
            <span className="chip">Go</span>
            <span className="chip">Tree-sitter</span>
            <span className="chip">SQLite</span>
            <span className="chip">React 19</span>
            <span className="chip">React Flow</span>
            <span className="chip">ELK</span>
          </div>
          <div className="copy">CodeAtlas — a determinism-first code map.</div>
        </div>
      </footer>
    </div>
  )
}
