export default function Login() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm rounded border border-border bg-surface p-8">
        <h1 className="text-xl font-semibold tracking-tight text-foreground">
          Issue Refiner
        </h1>
        <p className="mt-2 text-[13px] text-muted">
          Sign in with GitHub to browse your connected repositories and run
          evidence-backed investigations.
        </p>
        <a
          href="/auth/github"
          className="mt-6 block h-9 rounded bg-accent px-4 py-2 text-center text-[13px] font-medium text-white transition-colors hover:bg-accent-hover"
        >
          Sign in with GitHub
        </a>
      </div>
    </div>
  );
}
