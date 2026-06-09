"use client";

import clsx from "clsx";
import {
  Activity,
  Bell,
  Cable,
  ChevronDown,
  LayoutDashboard,
  LogOut,
  Menu,
  MessagesSquare,
  Monitor,
  Moon,
  Search,
  ShieldCheck,
  Sun,
  Users,
  User,
  Trash2,
  Workflow,
  X,
} from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useCallback, useEffect, useRef, useState, type MutableRefObject } from "react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useSidebar } from "@/lib/sidebar";
import { Theme, useTheme } from "@/lib/theme";
import type { ChatNotificationItem, ChatNotificationSummary } from "@/lib/types";
import { ProfileModal } from "@/components/profile-modal";

const navItems = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/chat", label: "Chat", icon: MessagesSquare },
  { href: "/chat/trash", label: "Thùng rác", icon: Trash2 },
  { href: "/admin/users", label: "Users", icon: Users },
  { href: "/admin/roles", label: "Roles", icon: ShieldCheck },
  { href: "/admin/teams", label: "Teams", icon: Workflow },
  { href: "/admin/channels", label: "Channels", icon: Cable },
  { href: "/admin/audit", label: "Audit", icon: Activity },
];

export function AdminShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { token, profile, loading, logout } = useAuth();
  const { isOpen, toggle, setIsOpen } = useSidebar();
  const { theme, resolvedTheme, setTheme } = useTheme();
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [notificationSummary, setNotificationSummary] = useState<ChatNotificationSummary>({ total_unread: 0, missed_count: 0, items: [] });
  const [toastItem, setToastItem] = useState<ChatNotificationItem | null>(null);
  const [profileOpen, setProfileOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const notificationsRef = useRef<HTMLDivElement>(null);
  const notificationsButtonRef = useRef<HTMLButtonElement>(null);
  const previousNotificationKeyRef = useRef("");
  const baseTitleRef = useRef("Omni Channel CMS");
  const faviconHrefRef = useRef<string | null>(null);

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
      if (
        notificationsRef.current &&
        !notificationsRef.current.contains(event.target as Node) &&
        notificationsButtonRef.current &&
        !notificationsButtonRef.current.contains(event.target as Node)
      ) {
        setNotificationsOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const loadNotifications = useCallback(
    async (quiet = false) => {
      if (!token) return;
      try {
        const summary = await api.chatNotifications(token, { limit: 20 });
        const nextKey = `${summary.latest_at || ""}:${summary.total_unread}:${summary.items[0]?.conversation_id || ""}`;
        const previousKey = previousNotificationKeyRef.current;
        setNotificationSummary(summary);
        updateBrowserBadge(summary.total_unread, baseTitleRef.current, faviconHrefRef);

        if (previousKey && nextKey !== previousKey && summary.total_unread > 0 && summary.items[0]) {
          setToastItem(summary.items[0]);
          window.setTimeout(() => setToastItem(null), 5000);
          if (document.hidden && Notification.permission === "granted") {
            showOSNotification(summary.items[0]);
          }
        }
        previousNotificationKeyRef.current = nextKey;
      } catch {
        if (!quiet) setNotificationSummary({ total_unread: 0, missed_count: 0, items: [] });
      }
    },
    [token],
  );

  useEffect(() => {
    if (!token || !profile) return;
    baseTitleRef.current = document.title || "Omni Channel CMS";
    faviconHrefRef.current = currentFaviconHref();
    void loadNotifications();
    const interval = window.setInterval(() => void loadNotifications(true), 6000);
    const handleChatRead = () => void loadNotifications(true);
    window.addEventListener("omni-chat-read", handleChatRead);
    return () => {
      window.clearInterval(interval);
      window.removeEventListener("omni-chat-read", handleChatRead);
      document.title = baseTitleRef.current;
      restoreFavicon(faviconHrefRef.current);
    };
  }, [loadNotifications, profile, token]);

  async function toggleNotifications() {
    const nextOpen = !notificationsOpen;
    setNotificationsOpen(nextOpen);
    if (nextOpen) {
      await maybeRequestNotificationPermission();
      void loadNotifications(true);
    }
  }

  function openNotification(item: ChatNotificationItem) {
    setNotificationsOpen(false);
    setToastItem(null);
    window.dispatchEvent(new CustomEvent("omni-open-conversation", { detail: item.conversation_id }));
    router.push(`/chat?conversation=${encodeURIComponent(item.conversation_id)}`);
  }

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
            {navItems.slice(0, 3).map((item) => (
              <NavLink
                key={item.href}
                item={item}
                pathname={pathname}
                isOpen={isOpen}
                setIsOpen={setIsOpen}
                hasUnread={item.href === "/chat" && notificationSummary.total_unread > 0}
              />
            ))}
          </div>
          <div className="space-y-1">
            {isOpen ? <h3 className="mb-2 px-3 text-[10px] font-bold uppercase tracking-wider" style={{ color: "var(--app-muted)" }}>System</h3> : null}
            {navItems.slice(3).map((item) => <NavLink key={item.href} item={item} pathname={pathname} isOpen={isOpen} setIsOpen={setIsOpen} />)}
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
            <div className="relative z-50" ref={notificationsRef}>
              <button
                ref={notificationsButtonRef}
                className="btn relative h-9 w-9 px-0"
                onClick={() => void toggleNotifications()}
                title="Notifications"
              >
                <Bell size={16} />
                {notificationSummary.total_unread > 0 ? (
                  <span className="absolute -right-1 -top-1 flex h-5 min-w-5 items-center justify-center rounded-full bg-red-600 px-1 text-[10px] font-bold text-white">
                    {notificationSummary.total_unread > 99 ? "99+" : notificationSummary.total_unread}
                  </span>
                ) : null}
              </button>
              {notificationsOpen ? (
                <NotificationMenu summary={notificationSummary} onOpen={openNotification} />
              ) : null}
            </div>
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
      {toastItem ? <NotificationToast item={toastItem} onOpen={openNotification} onClose={() => setToastItem(null)} /> : null}
    </div>
  );
}

function NavLink({
  item,
  pathname,
  isOpen,
  setIsOpen,
  hasUnread = false,
}: {
  item: (typeof navItems)[number];
  pathname: string;
  isOpen: boolean;
  setIsOpen: (open: boolean) => void;
  hasUnread?: boolean;
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
      <div className={clsx("relative flex items-center justify-center shrink-0", isOpen && "mr-3")}>
        <Icon className="h-5 w-5 shrink-0" style={{ color: active ? "var(--app-sidebar-item-active-text)" : "var(--app-sidebar-icon)" }} />
        {hasUnread ? (
          <span className="absolute -right-0.5 -top-0.5 h-2.5 w-2.5 rounded-full bg-red-600 ring-2 ring-[var(--app-sidebar-bg)]" />
        ) : null}
      </div>
      {isOpen ? <span>{item.label}</span> : null}
    </Link>
  );
}

function NotificationMenu({ summary, onOpen }: { summary: ChatNotificationSummary; onOpen: (item: ChatNotificationItem) => void }) {
  return (
    <div className="menu-motion absolute right-0 mt-2 w-80 overflow-hidden rounded-md border shadow-xl" style={{ background: "var(--app-surface)", borderColor: "var(--app-border)" }}>
      <div className="border-b p-3" style={{ borderColor: "var(--app-border)" }}>
        <div className="text-sm font-semibold">Tin nhắn mới / bỏ lỡ</div>
        <div className="mt-1 text-xs" style={{ color: "var(--app-muted)" }}>
          {summary.total_unread > 0 ? `${summary.total_unread} tin chưa đọc trong ${summary.missed_count} hội thoại` : "Không có tin nhắn bỏ lỡ"}
        </div>
      </div>
      <div className="max-h-96 overflow-y-auto p-2">
        {summary.items.length === 0 ? (
          <div className="px-3 py-8 text-center text-sm" style={{ color: "var(--app-muted)" }}>
            Bạn đã đọc hết tin nhắn.
          </div>
        ) : (
          summary.items.map((item) => (
            <button key={item.conversation_id} className="flex w-full gap-3 rounded-md px-3 py-2 text-left transition hover:bg-[var(--app-surface-hover)]" onClick={() => onOpen(item)}>
              <span className="mt-1 h-2.5 w-2.5 shrink-0 rounded-full bg-red-600" />
              <span className="min-w-0 flex-1">
                <span className="flex items-center justify-between gap-2">
                  <span className="truncate text-sm font-semibold">{item.customer_name || item.customer_ref || "Khách hàng"}</span>
                  <span className="shrink-0 rounded-full bg-red-600 px-2 py-0.5 text-[10px] font-bold text-white">{item.unread_count}</span>
                </span>
                <span className="mt-0.5 block truncate text-xs" style={{ color: "var(--app-muted)" }}>
                  {item.last_message_text || "(no text)"}
                </span>
                <span className="mt-1 block text-[11px]" style={{ color: "var(--app-muted)" }}>
                  {formatNotificationTime(item.last_message_at)}
                </span>
              </span>
            </button>
          ))
        )}
      </div>
    </div>
  );
}

function NotificationToast({ item, onOpen, onClose }: { item: ChatNotificationItem; onOpen: (item: ChatNotificationItem) => void; onClose: () => void }) {
  return (
    <div className="fixed right-4 top-20 z-[70] w-80 rounded-md border p-3 shadow-2xl" style={{ background: "var(--app-surface)", borderColor: "var(--app-border)" }}>
      <button className="absolute right-2 top-2 text-xs" style={{ color: "var(--app-muted)" }} onClick={onClose} title="Close">
        <X size={14} />
      </button>
      <button className="block w-full pr-4 text-left" onClick={() => onOpen(item)}>
        <div className="text-sm font-semibold">Tin nhắn mới từ {item.customer_name || item.customer_ref || "khách hàng"}</div>
        <div className="mt-1 truncate text-xs" style={{ color: "var(--app-muted)" }}>
          {item.last_message_text || "(no text)"}
        </div>
      </button>
    </div>
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

function formatNotificationTime(value: string) {
  if (!value) return "";
  return new Intl.DateTimeFormat("vi-VN", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

async function maybeRequestNotificationPermission() {
  if (typeof window === "undefined" || !("Notification" in window)) return;
  if (Notification.permission === "default") {
    await Notification.requestPermission().catch(() => undefined);
  }
}

function showOSNotification(item: ChatNotificationItem) {
  if (typeof window === "undefined" || !("Notification" in window) || Notification.permission !== "granted") return;
  const notification = new Notification(`Tin nhắn mới từ ${item.customer_name || item.customer_ref || "khách hàng"}`, {
    body: item.last_message_text || "Bạn có tin nhắn mới",
    tag: item.conversation_id,
  });
  window.setTimeout(() => notification.close(), 6000);
}

function currentFaviconHref() {
  return document.querySelector<HTMLLinkElement>('link[rel="icon"]')?.href || null;
}

function restoreFavicon(href: string | null) {
  const link = ensureFaviconLink();
  if (href) link.href = href;
}

function updateBrowserBadge(totalUnread: number, baseTitle: string, originalFaviconRef: MutableRefObject<string | null>) {
  document.title = totalUnread > 0 ? `(${totalUnread}) ${baseTitle}` : baseTitle;
  const link = ensureFaviconLink();
  if (totalUnread <= 0) {
    if (originalFaviconRef.current) link.href = originalFaviconRef.current;
    return;
  }
  const canvas = document.createElement("canvas");
  canvas.width = 32;
  canvas.height = 32;
  const context = canvas.getContext("2d");
  if (!context) return;
  context.fillStyle = "#0284c7";
  context.fillRect(0, 0, 32, 32);
  context.fillStyle = "#ffffff";
  context.font = "bold 18px sans-serif";
  context.textAlign = "center";
  context.textBaseline = "middle";
  context.fillText("O", 16, 16);
  context.fillStyle = "#dc2626";
  context.beginPath();
  context.arc(24, 8, 8, 0, Math.PI * 2);
  context.fill();
  context.fillStyle = "#ffffff";
  context.font = "bold 9px sans-serif";
  context.fillText(totalUnread > 9 ? "9+" : String(totalUnread), 24, 8);
  link.href = canvas.toDataURL("image/png");
}

function ensureFaviconLink() {
  let link = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
  if (!link) {
    link = document.createElement("link");
    link.rel = "icon";
    document.head.appendChild(link);
  }
  return link;
}
