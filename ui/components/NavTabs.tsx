"use client";

import Link from "next/link";
import { usePathname, useSearchParams, useRouter } from "next/navigation";
import { cn } from "@/lib/utils";

const tabs = [
  { href: "/", label: "Matchmaker" },
  { href: "/gameserver", label: "Game Server" },
];

export function NavTabs() {
  const path = usePathname();
  const params = useSearchParams();
  const router = useRouter();
  const demo = params.get("demo") === "1";

  function toggleDemo() {
    const next = new URLSearchParams(params.toString());
    if (demo) next.delete("demo");
    else next.set("demo", "1");
    router.replace(`${path}?${next.toString()}`);
  }

  return (
    <nav className="flex items-center gap-0 text-xs font-bold tracking-widest w-full">
      {tabs.map((t) => (
        <Link
          key={t.href}
          href={t.href}
          className={cn(
            "px-4 h-12 -mb-[2px] flex items-center text-xs tracking-widest cursor-pointer transition-colors border-b-2",
            path === t.href
              ? "text-zinc-100 border-violet-500"
              : "text-zinc-400 border-transparent hover:text-zinc-300 hover:border-zinc-600"
          )}
        >
          {t.label}
        </Link>
      ))}

      {/* Demo toggle — pushed to the right */}
      <button
        onClick={toggleDemo}
        className="ml-auto flex items-center gap-2 text-xs text-zinc-400 hover:text-zinc-200 transition-colors cursor-pointer"
        style={{ fontFamily: "var(--font-geist-mono)" }}
      >
        <span>DEMO</span>
        <span className={`relative inline-flex h-4 w-8 rounded-full transition-colors ${demo ? "bg-violet-600" : "bg-zinc-700"}`}>
          <span className={`absolute top-0.5 left-0.5 h-3 w-3 rounded-full bg-white transition-transform ${demo ? "translate-x-4" : "translate-x-0"}`} />
        </span>
      </button>
    </nav>
  );
}
