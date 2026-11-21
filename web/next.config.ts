import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone", // untuk image Docker
  cacheComponents: true,
  partialPrefetching: true,
};

export default nextConfig;
