"use client";

import clsx from "clsx";
import {
  Activity,
  Cable,
  ChevronDown,
  LayoutDashboard,
  LogOut,
  Menu,
  Monitor,
  Moon,
  Search,
  ShieldCheck,
  Sun,
  Users,
  User,
  Workflow,
} from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useAuth } from "@/lib/auth";
import { useSidebar } from "@/lib/sidebar";
import { Theme, useTheme } from "@/lib/theme";
import { ProfileModal } from "@/components/profile-modal";

const navItems = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/admin/users", label: "Users", icon: Users },
  { href: "/admin/roles", label: "Roles", icon: ShieldCheck },
  { href: "/admin/teams", label: "Teams", icon: Workflow },
  { href: "/admin/channels", label: "Channels", icon: Cable },
  { href: "/admin/audit", label: "Audit", icon: Activity },
];

export function AdminShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { profile, loading, logout } = useAuth();
  const { isOpen, toggle, setIsOpen } = useSidebar();
  const { theme, resolvedTheme, setTheme } = useTheme();
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [profileOpen, setProfileOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!loading && !profile) {
      router.replace("/login");
    }
  }, [loading, profile, router]);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setUserMenuOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  if (loading || !profile) {
    return <div className="flex min-h-screen items-center justify-center text-sm text-muted">Loading workspace...</div>;
  }

  return (
    <div className="flex min-h-screen" style={{ background: "var(--app-bg)" }}>
      {isOpen ? (
        <button className="fixed inset-0 z-40 bg-slate-950/30 lg:hidden" onClick={() => setIsOpen(false)} aria-label="Close sidebar" />
      ) : null}
      <aside
        className={clsx(
          "fixed inset-y-0 left-0 z-50 flex flex-col border-r transition-all duration-300 lg:static lg:z-auto lg:translate-x-0",
          isOpen ? "w-64 translate-x-0" : "w-64 -translate-x-full lg:w-16",
        )}
        style={{
          background: "var(--app-sidebar-bg)",
          borderColor: "var(--app-sidebar-border)",
          color: "var(--app-sidebar-text)",
        }}
      >
        <div className="flex h-16 items-center border-b px-4" style={{ borderColor: "var(--app-sidebar-border)" }}>
          {isOpen ? (
            <div className="flex min-w-0 flex-1 items-center gap-2">
              <div className="flex h-8 w-8 items-center justify-center rounded-md bg-sky-600 text-sm font-bold text-white">O</div>
              <div className="min-w-0">
                <div className="truncate text-sm font-semibold" style={{ color: "var(--app-sidebar-text-strong)" }}>
                  Omni Channel
                </div>
                <div className="truncate text-xs" style={{ color: "var(--app-muted)" }}>
                  Admin CMS
                </div>
              </div>
            </div>
          ) : null}
          <button className={clsx("btn h-9 w-9 px-0", !isOpen && "mx-auto")} onClick={toggle} title={isOpen ? "Close menu" : "Open menu"}>
            <Menu size={17} />
          </button>
        </div>
        <nav className="flex-1 space-y-5 overflow-y-auto px-3 py-4">
          <div className="space-y-1">
            {isOpen ? <h3 className="mb-2 px-3 text-[10px] font-bold uppercase tracking-wider" style={{ color: "var(--app-muted)" }}>Workspace</h3> : null}
            {navItems.slice(0, 1).map((item) => <NavLink key={item.href} item={item} pathname={pathname} isOpen={isOpen} setIsOpen={setIsOpen} />)}
          </div>
          <div className="space-y-1">
            {isOpen ? <h3 className="mb-2 px-3 text-[10px] font-bold uppercase tracking-wider" style={{ color: "var(--app-muted)" }}>System</h3> : null}
            {navItems.slice(1).map((item) => <NavLink key={item.href} item={item} pathname={pathname} isOpen={isOpen} setIsOpen={setIsOpen} />)}
          </div>
        </nav>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <header
          className="sticky top-0 z-30 flex h-16 items-center justify-between border-b px-4 shadow-sm backdrop-blur-md"
          style={{ background: "var(--app-topbar-bg)", borderColor: "var(--app-border)" }}
        >
          <div className="flex min-w-0 flex-1 items-center gap-3">
            <button className="btn h-9 w-9 px-0 lg:hidden" onClick={toggle} title={isOpen ? "Close menu" : "Open menu"}>
              <Menu size={17} />
            </button>
            <div className="relative hidden w-full max-w-md md:block">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2" style={{ color: "var(--app-muted)" }} />
              <input className="field h-9 pl-9" placeholder="Search users, roles, channels..." />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <ThemeSelector theme={theme} resolvedTheme={resolvedTheme} setTheme={setTheme} />
            <div className="relative z-50" ref={menuRef}>
              <button className="btn h-10 rounded-full px-2 pr-3" onClick={() => setUserMenuOpen(!userMenuOpen)}>
                <span className="flex h-7 w-7 items-center justify-center rounded-full bg-sky-100 text-sky-700">
                  <User size={15} />
                </span>
                <span className="hidden min-w-0 text-left sm:block">
                  <span className="block truncate text-xs font-bold">{profile.user.display_name}</span>
                  <span className="block truncate text-[10px] uppercase" style={{ color: "var(--app-muted)" }}>
                    {profile.user.email}
                  </span>
                </span>
                <ChevronDown className={clsx("h-3 w-3 transition-transform", userMenuOpen && "rotate-180")} />
              </button>
              {userMenuOpen ? (
                <div className="menu-motion absolute right-0 mt-2 w-60 overflow-hidden rounded-md border shadow-xl" style={{ background: "var(--app-surface)", borderColor: "var(--app-border)" }}>
                  <div className="border-b p-3" style={{ borderColor: "var(--app-border)" }}>
                    <p className="truncate text-sm font-semibold">{profile.user.display_name}</p>
                    <p className="truncate text-xs" style={{ color: "var(--app-muted)" }}>{profile.user.email}</p>
                  </div>
                  <div className="p-2">
                    <button className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm transition hover:bg-[var(--app-surface-hover)]" onClick={() => { setUserMenuOpen(false); setProfileOpen(true); }}>
                      <User size={15} /> Profile settings
                    </button>
                    <button className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-red-600 transition hover:bg-red-50" onClick={logout}>
                      <LogOut size={15} /> Logout
                    </button>
                  </div>
                </div>
              ) : null}
            </div>
          </div>
        </header>
        <main className="page-motion min-w-0 flex-1 p-4 lg:p-6">{children}</main>
      </div>
      <ProfileModal open={profileOpen} onClose={() => setProfileOpen(false)} />
    </div>
  );
}

function NavLink({
  item,
  pathname,
  isOpen,
  setIsOpen,
}: {
  item: (typeof navItems)[number];
  pathname: string;
  isOpen: boolean;
  setIsOpen: (open: boolean) => void;
}) {
  const Icon = item.icon;
  const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
  return (
    <Link
      href={item.href}
      onClick={() => {
        if (window.innerWidth < 1024) setIsOpen(false);
      }}
      title={!isOpen ? item.label : undefined}
      className={clsx(
        "group flex h-10 items-center rounded-md px-3 text-sm font-medium transition-colors",
        active ? "font-semibold" : "hover:bg-[var(--app-sidebar-item-hover-bg)] hover:text-[var(--app-sidebar-item-hover-text)]",
        !isOpen && "justify-center px-2",
      )}
      style={{
        background: active ? "var(--app-sidebar-item-active-bg)" : undefined,
        color: active ? "var(--app-sidebar-item-active-text)" : undefined,
      }}
    >
      <Icon className={clsx("h-5 w-5 shrink-0", isOpen && "mr-3")} style={{ color: active ? "var(--app-sidebar-item-active-text)" : "var(--app-sidebar-icon)" }} />
      {isOpen ? <span>{item.label}</span> : null}
    </Link>
  );
}

function ThemeSelector({
  theme,
  resolvedTheme,
  setTheme,
}: {
  theme: Theme;
  resolvedTheme: "light" | "dark";
  setTheme: (theme: Theme) => void;
}) {
  const [open, setOpen] = useState(false);
  const currentIcon = resolvedTheme === "dark" ? <Moon size={16} /> : <Sun size={16} />;
  const options: { value: Theme; label: string; icon: React.ReactNode }[] = [
    { value: "light", label: "Light", icon: <Sun size={15} /> },
    { value: "dark", label: "Dark", icon: <Moon size={15} /> },
    { value: "system", label: "System", icon: <Monitor size={15} /> },
  ];

  return (
    <div className="relative">
      <button className="btn h-9 w-9 px-0" onClick={() => setOpen(!open)} title="Theme">
        {currentIcon}
      </button>
      {open ? (
        <div className="menu-motion absolute right-0 mt-2 w-36 overflow-hidden rounded-md border shadow-xl" style={{ background: "var(--app-surface)", borderColor: "var(--app-border)" }}>
          {options.map((option) => (
            <button
              key={option.value}
              className={clsx("flex w-full items-center gap-2 px-3 py-2 text-sm transition hover:bg-[var(--app-surface-hover)]", theme === option.value && "font-semibold")}
              onClick={() => {
                setTheme(option.value);
                setOpen(false);
              }}
            >
              {option.icon}
              {option.label}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}
