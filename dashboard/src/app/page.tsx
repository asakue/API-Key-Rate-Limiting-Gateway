"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { getToken } from "@/lib/api";

export default function Home() {
  const router = useRouter();

  useEffect(() => {
    const token = getToken();
    if (token) {
      router.replace("/dashboard");
    } else {
      router.replace("/login");
    }
  }, [router]);

  return (
    <div className="flex h-screen items-center justify-center bg-slate-950 text-slate-400">
      <div className="animate-pulse flex items-center space-x-2">
        <span className="w-2.5 h-2.5 rounded-full bg-blue-500 animate-ping"></span>
        <span>Loading API Gateway Platform...</span>
      </div>
    </div>
  );
}

