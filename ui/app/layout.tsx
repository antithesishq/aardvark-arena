import type { Metadata } from "next";
import { Geist, Geist_Mono, Press_Start_2P, Silkscreen } from "next/font/google";
import "./globals.css";
import Link from "next/link";
import { NavTabs } from "@/components/NavTabs";

const geist = Geist({ subsets: ["latin"], variable: "--font-geist" });
const geistMono = Geist_Mono({ subsets: ["latin"], variable: "--font-geist-mono" });
const pressStart = Press_Start_2P({ subsets: ["latin"], weight: "400", variable: "--font-press-start" });
const silkscreen = Silkscreen({ subsets: ["latin"], weight: "400", variable: "--font-silkscreen" });

export const metadata: Metadata = {
  title: "Aardvark Arena",
  description: "Live dashboard for the Aardvark Arena distributed game system",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html
      lang="en"
      className={`dark h-full antialiased ${geist.variable} ${geistMono.variable} ${pressStart.variable} ${silkscreen.variable}`}
      style={{ fontFamily: "var(--font-geist), ui-sans-serif, system-ui, sans-serif" }}
    >
      <body className="min-h-full flex flex-col bg-zinc-950 text-zinc-100">
        {/* Top bar */}
        <header className="border-b border-zinc-800 bg-zinc-950">
          <div className="max-w-7xl mx-auto px-6 py-3 flex items-center gap-6">
            <Link href="/" style={{ fontFamily: "var(--font-press-start)" }} className="text-xs tracking-tight leading-none">
              <span className="text-zinc-100">AARDVARK</span>
              <span className="text-violet-400">ARENA</span>
            </Link>
            <NavTabs />
          </div>
        </header>
        <main className="flex-1 p-6">{children}</main>
      </body>
    </html>
  );
}
