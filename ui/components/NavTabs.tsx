"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";

const tabs = [
  { href: "/", label: "Matchmaker" },
  { href: "/gameserver", label: "Game Servers" },
];

export function NavTabs() {
  const path = usePathname();

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
              : "text-zinc-400 border-transparent hover:text-zinc-300 hover:border-zinc-600",
          )}
        >
          {t.label}
        </Link>
      ))}
    </nav>
  );
}
