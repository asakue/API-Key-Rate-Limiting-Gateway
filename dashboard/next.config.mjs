/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  env: {
    NEXT_PUBLIC_CONTROL_PLANE_URL: process.env.NEXT_PUBLIC_CONTROL_PLANE_URL || "http://localhost:8081",
    NEXT_PUBLIC_GATEWAY_URL: process.env.NEXT_PUBLIC_GATEWAY_URL || "http://localhost:8000",
  },
};

export default nextConfig;

