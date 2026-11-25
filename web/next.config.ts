import type { NextConfig } from "next";

// Address of the REST server as seen by the browser (also used by the CSP).
const serverApi = process.env.NEXT_PUBLIC_SERVER_API ?? "http://localhost:8000";

function originOf(url: string): string {
  try {
    return new URL(url).origin;
  } catch {
    return "";
  }
}

// Strict-ish CSP for production: same-origin scripts/workers only, WASM
// compilation allowed, network limited to this origin and the shop server.
// Next.js injects small inline bootstrap scripts, hence 'unsafe-inline' for
// scripts (no third-party script origins are allowed).
const csp = [
  "default-src 'self'",
  "script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval'",
  "worker-src 'self' blob:",
  `connect-src 'self' ${originOf(serverApi)}`.trim(),
  "img-src 'self' data: blob:",
  "style-src 'self' 'unsafe-inline'",
  "font-src 'self'",
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
  "frame-ancestors 'none'",
].join("; ");

const nextConfig: NextConfig = {
  output: "standalone", // untuk image Docker
  cacheComponents: true,
  partialPrefetching: true,
  async headers() {
    const common = [
      { key: "X-Content-Type-Options", value: "nosniff" },
      { key: "Referrer-Policy", value: "no-referrer" },
      { key: "Cross-Origin-Opener-Policy", value: "same-origin" },
      { key: "X-Frame-Options", value: "DENY" },
    ];
    if (process.env.NODE_ENV === "production") {
      common.push({ key: "Content-Security-Policy", value: csp });
    }
    return [{ source: "/:path*", headers: common }];
  },
};

export default nextConfig;
