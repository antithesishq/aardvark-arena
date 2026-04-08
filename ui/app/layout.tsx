import type { Metadata } from "next";
import { Geist, Geist_Mono, Silkscreen } from "next/font/google";
import "./globals.css";
import Link from "next/link";
import Image from "next/image";
import { NavTabs } from "@/components/NavTabs";

const geist = Geist({ subsets: ["latin"], variable: "--font-geist" });
const geistMono = Geist_Mono({ subsets: ["latin"], variable: "--font-geist-mono" });
const silkscreen = Silkscreen({ subsets: ["latin"], weight: "400", variable: "--font-silkscreen" });

export const metadata: Metadata = {
  title: "Aardvark Arena",
  description: "Live dashboard for the Aardvark Arena distributed game system",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html
      lang="en"
      className={`dark h-full antialiased ${geist.variable} ${geistMono.variable} ${silkscreen.variable}`}
      style={{ fontFamily: "var(--font-geist), ui-sans-serif, system-ui, sans-serif" }}
    >
      <body className="min-h-full flex flex-col bg-zinc-950 text-zinc-100">
        {/* Top bar */}
        <header className="sticky top-0 z-50 border-b border-zinc-800 bg-zinc-950/90 backdrop-blur-sm">
          <div className="max-w-7xl mx-auto flex items-center gap-8 h-12 px-6">
            <Link href="/" className="shrink-0">
              <Image src="/logo.avif" alt="Aardvark Arena" height={70} width={70} className="h-8 w-auto" priority />
            </Link>
            <NavTabs />
          </div>
        </header>
        <main className="flex-1 py-6 relative">
          <div
            className="fixed inset-0 bg-cover bg-center pointer-events-none -z-10"
            style={{ backgroundImage: "url('/background.avif')", backgroundAttachment: "fixed", opacity: 0.06 }}
          />
          <div className="relative z-10">{children}</div>
        </main>
      </body>
    </html>
  );
}
