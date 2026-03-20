"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";

const tabs = [
  { href: "/", label: "MATCHMAKER" },
  { href: "/gameserver", label: "GAME SERVER" },
];

export function NavTabs() {
  const path = usePathname();
  return (
    <nav className="flex gap-0 text-xs font-bold tracking-widest">
      {tabs.map((t) => (
        <Link
          key={t.href}
          href={t.href}
          style={{ fontFamily: "var(--font-silkscreen)" }}
          className={cn(
            "px-4 py-1.5 text-xs border-b-2 transition-colors tracking-widest cursor-pointer",
            path === t.href
              ? "text-zinc-100 border-violet-500"
              : "text-zinc-500 border-transparent hover:text-zinc-300 hover:border-zinc-600"
          )}
        >
          {t.label}
        </Link>
      ))}
    </nav>
  );
}
