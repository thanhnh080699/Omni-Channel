import type { Metadata } from "next";
import "./globals.css";
import { AuthProvider } from "@/lib/auth";
import { SidebarProvider } from "@/lib/sidebar";
import { ThemeProvider } from "@/lib/theme";

export const metadata: Metadata = {
  title: "Omni Channel CMS",
  description: "Administration CMS for Omni Channel platform",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <ThemeProvider>
          <AuthProvider>
            <SidebarProvider>{children}</SidebarProvider>
          </AuthProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
