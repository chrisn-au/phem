/** @type {import('next').NextConfig} */
const apiUrl = process.env.PHEM_API_URL_INTERNAL || "http://api:8080";

const nextConfig = {
  output: "standalone",
  reactStrictMode: true,
  // Proxy /api/* to the Go backend so the browser only knows about Next.js (port 3000).
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${apiUrl}/api/:path*`,
      },
    ];
  },
};

export default nextConfig;
