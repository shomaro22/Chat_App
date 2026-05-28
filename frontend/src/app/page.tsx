const profile = {
  name: 'Your Name',
  role: 'Frontend / Backend Engineer',
  summary:
    'ユーザーにとって迷いなく使えるUIと、運用しやすいWebアプリケーション設計を大切にしています。',
  location: 'Japan',
  email: 'hello@example.com',
  github: 'https://github.com/yourname',
};

const skills = [
  'Next.js',
  'TypeScript',
  'Tailwind CSS',
  'React',
  'Go',
  'Echo',
  'PostgreSQL',
  'Docker',
  'GitHub Actions',
];

const projects = [
  {
    title: 'Realtime Chat App',
    description:
      'WebSocketを利用したリアルタイムチャット。既読、リアクション、管理者機能まで含めて設計しました。',
    stack: ['React', 'Go', 'Echo', 'PostgreSQL', 'Docker'],
  },
  {
    title: 'Portfolio Site',
    description:
      'Next.jsとTailwind CSSで構築した静的ポートフォリオ。Vercelへのデプロイを想定しています。',
    stack: ['Next.js', 'TypeScript', 'Tailwind CSS', 'Vercel'],
  },
  {
    title: 'API Design Sample',
    description:
      'OpenAPIを起点にしたAPI設計サンプル。仕様、実装、テストの整合性を意識しています。',
    stack: ['OpenAPI', 'Swagger', 'Go', 'PostgreSQL'],
  },
];

const strengths = [
  '要件からUI、API、DBまで一貫して考えられること',
  '小さく動くものを作り、改善しやすい形に整理すること',
  'テスト、CI、ドキュメントを含めて再現性を担保すること',
];

export default function Home() {
  return (
    <main className="min-h-screen bg-slate-950 text-slate-100">
      <section className="relative isolate overflow-hidden">
        <div className="absolute inset-0 -z-10 bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.24),_transparent_34%),radial-gradient(circle_at_80%_20%,_rgba(168,85,247,0.2),_transparent_28%),linear-gradient(180deg,_#020617_0%,_#0f172a_100%)]" />
        <div className="mx-auto flex min-h-screen w-full max-w-6xl flex-col px-6 py-8 sm:px-10 lg:px-12">
          <header className="flex items-center justify-between">
            <a className="text-sm font-semibold tracking-[0.24em] text-cyan-300" href="#">
              PORTFOLIO
            </a>
            <nav aria-label="Main navigation" className="hidden gap-6 text-sm text-slate-300 sm:flex">
              <a className="hover:text-white" href="#about">
                About
              </a>
              <a className="hover:text-white" href="#projects">
                Projects
              </a>
              <a className="hover:text-white" href="#contact">
                Contact
              </a>
            </nav>
          </header>

          <div className="grid flex-1 items-center gap-12 py-20 lg:grid-cols-[1.1fr_0.9fr]">
            <div>
              <p className="mb-5 inline-flex rounded-full border border-cyan-400/30 bg-cyan-400/10 px-4 py-2 text-sm text-cyan-200">
                {profile.location} based web developer
              </p>
              <h1 className="max-w-3xl text-5xl font-bold tracking-tight text-white sm:text-6xl lg:text-7xl">
                {profile.name}
              </h1>
              <p className="mt-5 text-2xl font-semibold text-cyan-200">{profile.role}</p>
              <p className="mt-6 max-w-2xl text-lg leading-8 text-slate-300">{profile.summary}</p>
              <div className="mt-10 flex flex-col gap-4 sm:flex-row">
                <a
                  className="rounded-full bg-cyan-300 px-6 py-3 text-center text-sm font-bold text-slate-950 transition hover:bg-cyan-200"
                  href="#projects"
                >
                  制作物を見る
                </a>
                <a
                  className="rounded-full border border-white/15 px-6 py-3 text-center text-sm font-bold text-white transition hover:border-cyan-300/60 hover:text-cyan-200"
                  href={`mailto:${profile.email}`}
                >
                  連絡する
                </a>
              </div>
            </div>

            <aside className="rounded-3xl border border-white/10 bg-white/[0.04] p-6 shadow-2xl shadow-cyan-950/30 backdrop-blur">
              <p className="text-sm font-semibold uppercase tracking-[0.24em] text-slate-400">Tech Stack</p>
              <div className="mt-6 flex flex-wrap gap-3">
                {skills.map((skill) => (
                  <span
                    className="rounded-full border border-white/10 bg-slate-900/70 px-4 py-2 text-sm text-slate-200"
                    key={skill}
                  >
                    {skill}
                  </span>
                ))}
              </div>
            </aside>
          </div>
        </div>
      </section>

      <section className="bg-white px-6 py-24 text-slate-950 sm:px-10 lg:px-12" id="about">
        <div className="mx-auto grid max-w-6xl gap-12 lg:grid-cols-[0.8fr_1.2fr]">
          <div>
            <p className="text-sm font-bold uppercase tracking-[0.24em] text-cyan-600">About</p>
            <h2 className="mt-4 text-3xl font-bold tracking-tight sm:text-4xl">作るときに大切にしていること</h2>
          </div>
          <div className="grid gap-4">
            {strengths.map((strength) => (
              <div className="rounded-2xl border border-slate-200 bg-slate-50 p-5" key={strength}>
                <p className="text-lg font-semibold">{strength}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="bg-slate-100 px-6 py-24 text-slate-950 sm:px-10 lg:px-12" id="projects">
        <div className="mx-auto max-w-6xl">
          <div className="max-w-2xl">
            <p className="text-sm font-bold uppercase tracking-[0.24em] text-cyan-700">Projects</p>
            <h2 className="mt-4 text-3xl font-bold tracking-tight sm:text-4xl">制作物</h2>
            <p className="mt-4 text-slate-600">
              学習・個人開発・提出物として見せたい成果物を、課題と使用技術が伝わる形で整理します。
            </p>
          </div>

          <div className="mt-12 grid gap-6 lg:grid-cols-3">
            {projects.map((project) => (
              <article className="flex flex-col rounded-3xl bg-white p-6 shadow-sm ring-1 ring-slate-200" key={project.title}>
                <h3 className="text-xl font-bold">{project.title}</h3>
                <p className="mt-4 flex-1 leading-7 text-slate-600">{project.description}</p>
                <div className="mt-6 flex flex-wrap gap-2">
                  {project.stack.map((item) => (
                    <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-700" key={item}>
                      {item}
                    </span>
                  ))}
                </div>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="bg-slate-950 px-6 py-24 text-white sm:px-10 lg:px-12" id="contact">
        <div className="mx-auto max-w-4xl rounded-3xl border border-white/10 bg-white/[0.04] p-8 text-center sm:p-12">
          <p className="text-sm font-bold uppercase tracking-[0.24em] text-cyan-300">Contact</p>
          <h2 className="mt-4 text-3xl font-bold tracking-tight sm:text-4xl">お気軽にご連絡ください</h2>
          <p className="mx-auto mt-4 max-w-2xl leading-7 text-slate-300">
            案1ではバックエンドを持たせず、メールリンクと外部プロフィールへの導線に絞っています。
            問い合わせ保存や管理画面が必要になった段階で、案2のAPI構成へ拡張できます。
          </p>
          <div className="mt-8 flex flex-col justify-center gap-4 sm:flex-row">
            <a className="rounded-full bg-cyan-300 px-6 py-3 font-bold text-slate-950 hover:bg-cyan-200" href={`mailto:${profile.email}`}>
              {profile.email}
            </a>
            <a
              className="rounded-full border border-white/15 px-6 py-3 font-bold text-white hover:border-cyan-300/60 hover:text-cyan-200"
              href={profile.github}
              rel="noreferrer"
              target="_blank"
            >
              GitHub
            </a>
          </div>
        </div>
      </section>
    </main>
  );
}
