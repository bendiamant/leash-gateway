import type { Metadata } from "next";
import { Inter } from "next/font/google";
import { Toaster } from "@/components/ui/sonner";
import "./globals.css";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Leash Security Gateway - LLM Security & Governance Demo",
  description: "Experience enterprise-grade LLM security and governance with the Leash Security Gateway. Zero code changes required.",
  keywords: "LLM security, AI governance, API gateway, OpenAI, Anthropic, Google AI, enterprise security",
  authors: [{ name: "Leash Security" }],
  openGraph: {
    title: "Leash Security Gateway Demo",
    description: "Enterprise LLM Security & Governance Platform",
    type: "website",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={inter.className}>
        {children}
        <Toaster position="bottom-right" />
      </body>
    </html>
  );
}
