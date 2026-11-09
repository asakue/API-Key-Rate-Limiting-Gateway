import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "API Gateway & Rate Limiting Control Plane",
  description: "Ultra-low-latency API Key Management & Rate Limiting Reverse Proxy Platform",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body className="min-h-screen bg-slate-950 text-slate-100 antialiased selection:bg-blue-600 selection:text-white">
        {children}
      </body>
    </html>
  );
}

