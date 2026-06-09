"use client";

import clsx from "clsx";
import {
  AlertTriangle,
  Check,
  CheckCheck,
  Clock3,
  EyeOff,
  Hash,
  Info,
  Loader2,
  Mail,
  MessageCircle,
  MoreHorizontal,
  Pin,
  RefreshCw,
  Search,
  Send,
  Tag,
  Timer,
  Trash2,
  User,
  Users,
  VolumeX,
  X,
} from "lucide-react";
import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { AdminShell } from "@/components/admin-shell";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { Channel, ChannelAccount, Conversation, Message, TypingStatus } from "@/lib/types";

type Scope = "my" | "team";
type DetailTab = "info" | "tags" | "history";

const pollIntervalMs = 6000;

export default function ChatPage() {
  const { token } = useAuth();
  const [scope, setScope] = useState<Scope>("my");
  const [query, setQuery] = useState("");
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [messages, setMessages] = useState<Message[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [accounts, setAccounts] = useState<ChannelAccount[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [loadingList, setLoadingList] = useState(true);
  const [loadingMessages, setLoadingMessages] = useState(false);
  const [sending, setSending] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [error, setError] = useState("");
  const [composer, setComposer] = useState("");
  const [composing, setComposing] = useState(false);
  const [activeTab, setActiveTab] = useState<DetailTab>("info");
  const [tagDraft, setTagDraft] = useState("");
  const [savingTags, setSavingTags] = useState(false);
  const [typingStatus, setTypingStatus] = useState<TypingStatus>({ typing: false });
  const [requestedConversationId, setRequestedConversationId] = useState("");
  const [selectedAccountID, setSelectedAccountID] = useState("");
  const [creatingConv, setCreatingConv] = useState(false);
  const [createError, setCreateError] = useState("");

  const [searchTab, setSearchTab] = useState("Tất cả");
  const [phoneChecking, setPhoneChecking] = useState(false);
  const [validAccounts, setValidAccounts] = useState<Record<string, boolean>>({});

  useEffect(() => {
    if (query.trim() && accounts.length > 0 && !selectedAccountID) {
      setSelectedAccountID(accounts[0].id);
    }
  }, [query, accounts, selectedAccountID]);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const accountById = useMemo(() => new Map(accounts.map((account) => [account.id, account])), [accounts]);
  const channelById = useMemo(() => new Map(channels.map((channel) => [channel.id, channel])), [channels]);
  const selectedConversation = useMemo(
    () => conversations.find((conversation) => conversation.id === selectedId) || null,
    [conversations, selectedId],
  );

  const [avatarUrls, setAvatarUrls] = useState<Record<string, string>>({});
  const fetchingRefs = useRef<Set<string>>(new Set());

  useEffect(() => {
    if (!token || conversations.length === 0 || accounts.length === 0 || channels.length === 0) return;

    conversations.forEach(async (conv) => {
      const account = accountById.get(conv.channel_account_id);
      const channel = account ? channelById.get(account.channel_id) : undefined;
      if (channel?.code === "whatsapp" && conv.customer_ref) {
        const key = `${conv.channel_account_id}:${conv.customer_ref}`;
        if (avatarUrls[key] === undefined && !fetchingRefs.current.has(key)) {
          fetchingRefs.current.add(key);
          try {
            const res = await api.getWhatsAppAvatar(token, conv.channel_account_id, conv.customer_ref);
            if (res.url) {
              setAvatarUrls((prev) => ({ ...prev, [key]: res.url }));
            } else {
              setAvatarUrls((prev) => ({ ...prev, [key]: "" }));
            }
          } catch {
            setAvatarUrls((prev) => ({ ...prev, [key]: "" }));
          }
        }
      }
    });
  }, [conversations, token, accounts, channels, accountById, channelById, avatarUrls]);

  useEffect(() => {
    const digitsOnly = query.trim().replace(/[^0-9]/g, "");
    const isPhone = digitsOnly.length >= 9 && digitsOnly.length <= 15;

    if (!isPhone || !token || accounts.length === 0) {
      setValidAccounts({});
      setPhoneChecking(false);
      return;
    }

    setPhoneChecking(true);
    const delayDebounce = setTimeout(async () => {
      try {
        const results: Record<string, boolean> = {};
        await Promise.all(
          accounts.map(async (account) => {
            const channel = channelById.get(account.channel_id);
            if (channel?.code === "whatsapp") {
              try {
                const res = await api.checkChannelAccountPhone(token, account.id, query.trim());
                results[account.id] = res.exists;
              } catch {
                results[account.id] = false;
              }
            } else {
              results[account.id] = true;
            }
          })
        );
        setValidAccounts(results);
      } catch {
        setValidAccounts({});
      } finally {
        setPhoneChecking(false);
      }
    }, 500);

    return () => clearTimeout(delayDebounce);
  }, [query, token, accounts, channelById]);
  const visibleConversations = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    return conversations.filter((conversation) => {
      const account = accountById.get(conversation.channel_account_id);
      const channel = account ? channelById.get(account.channel_id) : undefined;
      const haystack = [
        conversation.customer_name,
        displayCustomer(conversation),
        conversation.customer_ref,
        conversation.external_conversation_id,
        account?.name,
        channel?.name,
        ...(conversation.tags || []),
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return !normalizedQuery || haystack.includes(normalizedQuery);
    });
  }, [accountById, channelById, conversations, query]);

  const loadConversations = useCallback(
    async (quiet = false) => {
      if (!token) return;
      if (quiet) setSyncing(true);
      else setLoadingList(true);
      try {
        const [conversationResult, channelResult, accountResult] = await Promise.all([
          api.conversations(token, scope, { limit: 100 }),
          api.chatChannels(token),
          api.chatChannelAccounts(token),
        ]);
        const nextConversations = conversationResult.data || [];
        setConversations(nextConversations);
        setChannels(channelResult.data || []);
        setAccounts(accountResult.data || []);
        setSelectedId((current) => current || nextConversations[0]?.id || "");
        setError("");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not load conversations");
      } finally {
        setLoadingList(false);
        setSyncing(false);
      }
    },
    [scope, token],
  );

  const loadMessages = useCallback(
    async (conversationId: string, quiet = false) => {
      if (!token || !conversationId) return;
      if (!quiet) setLoadingMessages(true);
      try {
        const result = await api.conversationMessages(token, conversationId, { limit: 100 });
        const nextMessages = result.data || [];
        setMessages(nextMessages);
        const lastMessage = nextMessages[nextMessages.length - 1];
        if (lastMessage) {
          await api.markConversationRead(token, conversationId, lastMessage.id);
          window.dispatchEvent(new CustomEvent("omni-chat-read"));
        }
        setError("");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not load messages");
      } finally {
        setLoadingMessages(false);
      }
    },
    [token],
  );

  useEffect(() => {
    void loadConversations();
  }, [loadConversations]);

  useEffect(() => {
    function readRequestedConversation() {
      setRequestedConversationId(new URLSearchParams(window.location.search).get("conversation") || "");
    }
    function handleOpenConversation(event: Event) {
      const conversationId = (event as CustomEvent<string>).detail;
      if (conversationId) setRequestedConversationId(conversationId);
    }
    readRequestedConversation();
    window.addEventListener("popstate", readRequestedConversation);
    window.addEventListener("omni-open-conversation", handleOpenConversation);
    return () => {
      window.removeEventListener("popstate", readRequestedConversation);
      window.removeEventListener("omni-open-conversation", handleOpenConversation);
    };
  }, []);

  useEffect(() => {
    if (requestedConversationId && conversations.some((conversation) => conversation.id === requestedConversationId)) {
      setSelectedId(requestedConversationId);
    }
  }, [conversations, requestedConversationId]);

  useEffect(() => {
    if (!selectedId) {
      setMessages([]);
      return;
    }
    void loadMessages(selectedId);
  }, [loadMessages, selectedId]);

  useEffect(() => {
    if (!token) return;
    const interval = window.setInterval(() => {
      void loadConversations(true);
      if (selectedId) void loadMessages(selectedId, true);
    }, pollIntervalMs);
    return () => window.clearInterval(interval);
  }, [loadConversations, loadMessages, selectedId, token]);

  useEffect(() => {
    if (!token || !selectedId) {
      setTypingStatus({ typing: false });
      return;
    }
    let cancelled = false;
    async function loadTyping() {
      try {
        const status = await api.conversationTyping(token!, selectedId);
        if (!cancelled) setTypingStatus(status);
      } catch {
        if (!cancelled) setTypingStatus({ typing: false });
      }
    }
    void loadTyping();
    const interval = window.setInterval(() => void loadTyping(), 2000);
    return () => {
      cancelled = true;
      window.clearInterval(interval);
    };
  }, [selectedId, token]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [messages.length, selectedId]);

  useEffect(() => {
    if (!selectedId && visibleConversations[0]) {
      setSelectedId(visibleConversations[0].id);
    }
  }, [selectedId, visibleConversations]);

  async function handleCreateDirectConversation(accountId: string, customerRef: string) {
    if (!token || !customerRef || !accountId) return;
    setCreatingConv(true);
    setCreateError("");
    try {
      const result = await api.createConversation(token, {
        channel_account_id: accountId,
        customer_ref: customerRef,
      });
      const newConv = result.data;
      setConversations((current) => {
        if (current.some((c) => c.id === newConv.id)) return current;
        return [newConv, ...current];
      });
      setSelectedId(newConv.id);
      setQuery(""); // Clear the search query to show all chats
      setSelectedAccountID("");
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : "Could not create conversation");
    } finally {
      setCreatingConv(false);
    }
  }

  async function sendMessage(event: FormEvent) {
    event.preventDefault();
    if (!token || !selectedConversation || !composer.trim()) return;
    setSending(true);
    try {
      const text = composer.trim();
      setComposer("");
      const result = await api.sendConversationMessage(token, selectedConversation.id, text);
      setMessages((current) => [...current, result.data]);
      await loadConversations(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not send message");
    } finally {
      setSending(false);
    }
  }

  async function handleDeleteConversation(id: string) {
    if (!token) return;
    if (!window.confirm("Bạn có chắc chắn muốn xoá cuộc hội thoại này? Cuộc hội thoại sẽ được đưa vào Thùng rác.")) return;
    try {
      await api.deleteConversation(token, id);
      setConversations((current) => current.filter((c) => c.id !== id));
      setSelectedId("");
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not delete conversation");
    }
  }

  async function saveTags(nextTags: string[]) {
    if (!token || !selectedConversation) return;
    setSavingTags(true);
    try {
      const result = await api.updateConversationTags(token, selectedConversation.id, nextTags);
      setConversations((current) => current.map((item) => (item.id === result.data.id ? result.data : item)));
      setTagDraft("");
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not update tags");
    } finally {
      setSavingTags(false);
    }
  }

  function addTag(event: FormEvent) {
    event.preventDefault();
    const tag = tagDraft.trim();
    if (!selectedConversation || !tag) return;
    void saveTags([...(selectedConversation.tags || []), tag]);
  }

  function channelLabel(conversation: Conversation) {
    const account = accountById.get(conversation.channel_account_id);
    const channel = account ? channelById.get(account.channel_id) : undefined;
    return {
      accountName: account?.name || "Unknown account",
      channelName: channel?.name || "Unknown channel",
      channelCode: channel?.code || "unknown",
    };
  }

  return (
    <AdminShell>
      {error && error.includes("Mất kết nối") && (
        <div className="mb-3 rounded-md bg-red-50 border border-red-200 px-4 py-2.5 text-sm text-red-700 flex items-center gap-2 shadow-sm animate-pulse">
          <Loader2 className="h-4 w-4 animate-spin text-red-700" />
          <span className="font-medium">{error}</span>
        </div>
      )}
      <div className="flex h-[calc(100vh-7rem)] min-h-[680px] flex-col overflow-hidden rounded-md border shadow-sm lg:flex-row" style={{ background: "var(--app-surface)", borderColor: "var(--app-border)" }}>
        <aside className="flex min-h-0 border-b lg:w-80 lg:flex-col lg:border-b-0 lg:border-r" style={{ borderColor: "var(--app-border)" }}>
          <div className="flex min-h-0 w-full flex-col">
            <div className="border-b p-3" style={{ borderColor: "var(--app-border)" }}>
              <div className="mb-3 flex items-center justify-between gap-2">
                <div className="flex items-center gap-2 font-semibold">
                  <MessageCircle className="h-5 w-5 text-accent" />
                  Chat
                </div>
                <button className="btn h-8 w-8 px-0" title="Sync conversations" onClick={() => void loadConversations(true)}>
                  {syncing ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
                </button>
              </div>
              <div className="mb-3 flex items-center gap-2">
                <div className="relative flex-1">
                  <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2" style={{ color: "var(--app-muted)" }} />
                  <input className="field h-9 pl-9" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search conversations..." />
                  {query && (
                    <button
                      type="button"
                      onClick={() => {
                        setQuery("");
                        setValidAccounts({});
                        setPhoneChecking(false);
                      }}
                      className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 hover:bg-black/10 rounded-full"
                    >
                      <X size={14} style={{ color: "var(--app-muted)" }} />
                    </button>
                  )}
                </div>
                {query && (
                  <button
                    type="button"
                    onClick={() => {
                      setQuery("");
                      setSelectedAccountID("");
                      setValidAccounts({});
                      setPhoneChecking(false);
                      setSearchTab("Tất cả");
                    }}
                    className="text-xs font-semibold shrink-0 text-sky-600 hover:text-sky-700 transition"
                  >
                    Đóng
                  </button>
                )}
              </div>
              {query.trim() && (
                <div className="mb-3">
                  <div className="mb-3 flex gap-4 border-b" style={{ borderColor: "var(--app-border)" }}>
                    {["Tất cả", "Liên hệ", "Tin nhắn"].map((tab) => (
                      <button
                        key={tab}
                        type="button"
                        onClick={() => setSearchTab(tab)}
                        className={clsx(
                          "pb-2 border-b-2 text-xs font-bold transition",
                          searchTab === tab
                            ? "border-sky-600 text-sky-600"
                            : "border-transparent text-[var(--app-muted-strong)]"
                        )}
                      >
                        {tab}
                      </button>
                    ))}
                  </div>
                  {(searchTab === "Tất cả" || searchTab === "Liên hệ") && (
                    <>
                      <div className="text-xs font-bold mb-2" style={{ color: "var(--app-text)" }}>
                        Tìm bạn qua số điện thoại:
                      </div>
                      <div className="space-y-1 max-h-48 overflow-y-auto">
                        {phoneChecking ? (
                          <div className="flex items-center gap-2 py-3 px-2 text-xs" style={{ color: "var(--app-muted)" }}>
                            <Loader2 className="h-3.5 w-3.5 animate-spin" />
                            Đang kiểm tra tài khoản WhatsApp...
                          </div>
                        ) : (() => {
                          const filteredAccounts = accounts.filter((account) => {
                            const channel = channelById.get(account.channel_id);
                            if (channel?.code === "whatsapp") {
                              return validAccounts[account.id] === true;
                            }
                            return true;
                          });

                          if (filteredAccounts.length === 0) {
                            return (
                              <div className="py-3 px-2 text-xs" style={{ color: "var(--app-muted)" }}>
                                Không tìm thấy liên hệ nào trên WhatsApp đăng ký số điện thoại này.
                              </div>
                            );
                          }

                          return filteredAccounts.map((account) => {
                            const channel = channelById.get(account.channel_id);
                            const cleanQuery = query.trim().replace(/[^0-9]/g, "");
                            const normQuery = cleanQuery.startsWith("0") ? "84" + cleanQuery.substring(1) : cleanQuery;
                            const matchedConv = conversations.find((c) => {
                              const cleanRef = (c.customer_ref || "").replace(/[^0-9]/g, "");
                              const normRef = cleanRef.startsWith("0") ? "84" + cleanRef.substring(1) : cleanRef;
                              const cleanExt = (c.external_conversation_id || "").split("@")[0].replace(/[^0-9]/g, "");
                              const normExt = cleanExt.startsWith("0") ? "84" + cleanExt.substring(1) : cleanExt;
                              return normRef === normQuery || normExt === normQuery;
                            });
                            const displayName = matchedConv ? displayCustomer(matchedConv) : `Khách hàng (${channel?.name || "Kênh"})`;
                            const isCat = displayName.includes("Số 2");
                            return (
                              <button
                                key={account.id}
                                type="button"
                                disabled={creatingConv}
                                onClick={() => handleCreateDirectConversation(account.id, query.trim())}
                                className="w-full text-left py-2 px-1 flex items-center gap-3 transition hover:bg-[var(--app-surface-hover)] border-b"
                                style={{ borderColor: "var(--app-border)" }}
                              >
                                {isCat ? (
                                  <img
                                    src="https://images.unsplash.com/photo-1514888286974-6c03e2ca1dba?w=100&h=100&fit=crop&q=80"
                                    alt="Avatar"
                                    className="h-10 w-10 shrink-0 rounded-full object-cover border"
                                    style={{ borderColor: "var(--app-border)" }}
                                  />
                                ) : (
                                  <div className="relative flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[var(--app-surface-muted)] font-bold text-xs border" style={{ borderColor: "var(--app-border)" }}>
                                    {initials(displayName)}
                                  </div>
                                )}
                                <div className="min-w-0 flex-1">
                                  <div className="truncate text-xs font-bold" style={{ color: "var(--app-text)" }}>
                                    {displayName}
                                  </div>
                                  <div className="truncate text-[11px]" style={{ color: "var(--app-muted)" }}>
                                    Số điện thoại: {query.trim()}
                                  </div>
                                  <div className="truncate text-[9px] uppercase font-bold text-sky-600 mt-0.5">
                                    {channel?.name || "Kênh"} · {account.name}
                                  </div>
                                </div>
                              </button>
                            );
                          });
                        })()}
                      </div>
                    </>
                  )}
                  {searchTab === "Tin nhắn" && (
                    <div className="p-6 text-center text-xs" style={{ color: "var(--app-muted)" }}>
                      Không có tin nhắn nào khớp với từ khóa tìm kiếm.
                    </div>
                  )}
                  {createError && (
                    <div className="text-[10px] text-red-600 font-semibold mt-1">{createError}</div>
                  )}
                </div>
              )}
              <div className="mb-3 flex rounded-lg bg-[var(--app-surface-muted)] p-1">
                <button
                  type="button"
                  className={clsx(
                    "flex-1 rounded-md py-1.5 text-center text-sm font-medium transition-all duration-200",
                    scope === "my"
                      ? "bg-[var(--app-surface)] text-[var(--app-text)] shadow-sm"
                      : "text-[var(--app-muted-strong)] hover:text-[var(--app-text)]"
                  )}
                  onClick={() => setScope("my")}
                >
                  My chats
                </button>
                <button
                  type="button"
                  className={clsx(
                    "flex-1 rounded-md py-1.5 text-center text-sm font-medium transition-all duration-200",
                    scope === "team"
                      ? "bg-[var(--app-surface)] text-[var(--app-text)] shadow-sm"
                      : "text-[var(--app-muted-strong)] hover:text-[var(--app-text)]"
                  )}
                  onClick={() => setScope("team")}
                >
                  Team chats
                </button>
              </div>
            </div>
            <div className="min-h-0 flex-1 overflow-y-auto">
              {loadingList ? (
                <div className="flex h-48 items-center justify-center">
                  <Loader2 className="h-5 w-5 animate-spin" style={{ color: "var(--app-muted)" }} />
                </div>
              ) : visibleConversations.length === 0 ? (
                <div className="p-6 text-center text-sm" style={{ color: "var(--app-muted)" }}>
                  No conversations match this view.
                </div>
              ) : (
                visibleConversations.map((conversation) => {
                  const key = `${conversation.channel_account_id}:${conversation.customer_ref}`;
                  return (
                    <ConversationRow
                      key={conversation.id}
                      conversation={conversation}
                      selected={conversation.id === selectedId}
                      labels={channelLabel(conversation)}
                      onClick={() => setSelectedId(conversation.id)}
                      onDelete={() => handleDeleteConversation(conversation.id)}
                      avatarUrl={avatarUrls[key]}
                    />
                  );
                })
              )}
            </div>
          </div>
        </aside>

        <section className="flex min-h-0 flex-1 flex-col">
          {selectedConversation ? (
            <>
              <div className="flex min-h-16 items-center justify-between gap-3 border-b px-4 py-3" style={{ borderColor: "var(--app-border)" }}>
                <div className="min-w-0">
                  <div className="flex min-w-0 items-center gap-2">
                    <h1 className="truncate text-base font-semibold">{displayCustomer(selectedConversation)}</h1>
                  </div>
                  <div className="mt-1 flex flex-wrap items-center gap-2 text-xs" style={{ color: "var(--app-muted)" }}>
                    <span>{channelLabel(selectedConversation).channelName}</span>
                    <span>·</span>
                    <span>{channelLabel(selectedConversation).accountName}</span>
                    <span>·</span>
                    <span>{selectedConversation.external_conversation_id}</span>
                  </div>
                </div>
              </div>
              <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5" style={{ background: "var(--app-bg)" }}>
                {loadingMessages ? (
                  <div className="flex h-full items-center justify-center">
                    <Loader2 className="h-6 w-6 animate-spin" style={{ color: "var(--app-muted)" }} />
                  </div>
                ) : messages.length === 0 && !typingStatus.typing ? (
                  <div className="flex h-full items-center justify-center text-sm" style={{ color: "var(--app-muted)" }}>
                    No messages yet.
                  </div>
                ) : (
                  <div className="mx-auto flex max-w-3xl flex-col gap-3">
                    {messages.map((message) => (
                      <MessageBubble key={message.id} message={message} senderName={displayCustomer(selectedConversation)} />
                    ))}
                    {typingStatus.typing ? <TypingIndicator senderName={displayCustomer(selectedConversation)} /> : null}
                    <div ref={messagesEndRef} />
                  </div>
                )}
              </div>
              <form className="border-t p-3" style={{ borderColor: "var(--app-border)" }} onSubmit={sendMessage}>
                {error && !error.includes("Mất kết nối") ? (
                  <div className="mb-2 rounded-md bg-red-50 px-3 py-2 text-sm text-red-700 flex items-center gap-2">
                    <span>{error}</span>
                  </div>
                ) : null}
                <div className="flex items-end gap-2">
                  <textarea
                    className="field min-h-11 resize-none py-2"
                    rows={1}
                    value={composer}
                    onChange={(event) => setComposer(event.target.value)}
                    onCompositionStart={() => setComposing(true)}
                    onCompositionEnd={() => setComposing(false)}
                    onKeyDown={(event) => {
                      const nativeEvent = event.nativeEvent as KeyboardEvent & { isComposing?: boolean };
                      if (composing || nativeEvent.isComposing || nativeEvent.keyCode === 229) return;
                      if (event.key === "Enter" && !event.shiftKey) {
                        event.preventDefault();
                        event.currentTarget.form?.requestSubmit();
                      }
                    }}
                    placeholder="Reply to customer..."
                  />
                  <button className="btn btn-primary h-11 w-11 px-0" disabled={sending || !composer.trim()} title="Send message">
                    {sending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
                  </button>
                </div>
              </form>
            </>
          ) : (
            <div className="flex h-full items-center justify-center text-sm" style={{ color: "var(--app-muted)" }}>
              Select a conversation to start.
            </div>
          )}
        </section>

        <aside className="min-h-0 border-t lg:w-80 lg:border-l lg:border-t-0" style={{ borderColor: "var(--app-border)" }}>
          {selectedConversation ? (
            <div className="flex h-full min-h-0 flex-col">
              <div className="border-b p-4" style={{ borderColor: "var(--app-border)" }}>
                <div className="flex items-center gap-3">
                  <div className="flex h-11 w-11 items-center justify-center rounded-md bg-[var(--app-accent-soft-bg)] text-[var(--app-accent-soft-fg)]">
                    <User size={19} />
                  </div>
                  <div className="min-w-0">
                    <div className="truncate font-semibold">{displayCustomer(selectedConversation)}</div>
                    <div className="truncate text-xs" style={{ color: "var(--app-muted)" }}>
                      {channelLabel(selectedConversation).channelName}
                    </div>
                  </div>
                </div>
              </div>
              <div className="grid grid-cols-3 border-b text-sm" style={{ borderColor: "var(--app-border)" }}>
                {[
                  { id: "info" as const, label: "Thông tin", icon: Info },
                  { id: "tags" as const, label: "Tags", icon: Tag },
                  { id: "history" as const, label: "Lịch sử", icon: Clock3 },
                ].map((tab) => {
                  const Icon = tab.icon;
                  return (
                    <button
                      key={tab.id}
                      className={clsx("flex h-11 items-center justify-center gap-1 border-b-2 text-xs font-medium", activeTab === tab.id ? "border-sky-600 text-sky-700" : "border-transparent")}
                      onClick={() => setActiveTab(tab.id)}
                    >
                      <Icon size={14} />
                      {tab.label}
                    </button>
                  );
                })}
              </div>
              <div className="min-h-0 flex-1 overflow-y-auto p-4">
                {activeTab === "info" ? (
                  <InfoTab conversation={selectedConversation} labels={channelLabel(selectedConversation)} />
                ) : activeTab === "tags" ? (
                  <TagsTab
                    conversation={selectedConversation}
                    draft={tagDraft}
                    saving={savingTags}
                    setDraft={setTagDraft}
                    onAdd={addTag}
                    onRemove={(tag) => void saveTags((selectedConversation.tags || []).filter((item) => item !== tag))}
                  />
                ) : (
                  <HistoryTab conversation={selectedConversation} messages={messages} />
                )}
              </div>
            </div>
          ) : (
            <div className="p-6 text-center text-sm" style={{ color: "var(--app-muted)" }}>
              Customer details appear after selecting a conversation.
            </div>
          )}
        </aside>
      </div>
    </AdminShell>
  );
}

function ConversationRow({
  conversation,
  labels,
  selected,
  onClick,
  onDelete,
  avatarUrl,
}: {
  conversation: Conversation;
  labels: { accountName: string; channelName: string; channelCode: string };
  selected: boolean;
  onClick: () => void;
  onDelete: () => void;
  avatarUrl?: string;
}) {
  const unread = Boolean(conversation.has_unread || conversation.unread_count > 0);
  const [showDropdown, setShowDropdown] = useState(false);
  const [dropdownPosition, setDropdownPosition] = useState({ top: 0, left: 0 });
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const updateDropdownPosition = useCallback(() => {
    const trigger = menuButtonRef.current;
    if (!trigger) return;

    const triggerRect = trigger.getBoundingClientRect();
    const menuWidth = 208;
    const estimatedMenuHeight = 280;
    const viewportPadding = 12;
    const preferredTop = triggerRect.bottom + 8;
    const maxTop = window.innerHeight - estimatedMenuHeight - viewportPadding;

    setDropdownPosition({
      top: Math.max(viewportPadding, Math.min(preferredTop, maxTop)),
      left: Math.max(viewportPadding, triggerRect.right - menuWidth),
    });
  }, []);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      const target = event.target as Node;
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(target) &&
        !menuButtonRef.current?.contains(target)
      ) {
        setShowDropdown(false);
      }
    }
    if (showDropdown) {
      updateDropdownPosition();
      document.addEventListener("mousedown", handleClickOutside);
      window.addEventListener("resize", updateDropdownPosition);
      window.addEventListener("scroll", updateDropdownPosition, true);
    }
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      window.removeEventListener("resize", updateDropdownPosition);
      window.removeEventListener("scroll", updateDropdownPosition, true);
    };
  }, [showDropdown, updateDropdownPosition]);

  const displayName = displayCustomer(conversation);
  const isCat = displayName.includes("Số 2");

  return (
    <div 
      className="relative group"
      style={{ zIndex: showDropdown ? 100 : 1 }}
    >
      <button
        className={clsx(
          "flex w-full gap-3 border-b p-3 text-left transition hover:bg-[var(--app-surface-hover)]",
          selected && "bg-[var(--app-accent-soft-bg)]",
          unread && !selected && "bg-red-50/50"
        )}
        style={{ borderColor: "var(--app-border)" }}
        onClick={onClick}
      >
        <div 
          className="relative flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-[var(--app-surface-muted)] font-semibold border"
          style={{ borderColor: "var(--app-border)" }}
        >
          {avatarUrl ? (
            <img src={avatarUrl} alt="Avatar" className="h-full w-full rounded-md object-cover" />
          ) : isCat ? (
            <img
              src="https://images.unsplash.com/photo-1514888286974-6c03e2ca1dba?w=100&h=100&fit=crop&q=80"
              alt="Avatar"
              className="h-full w-full rounded-md object-cover"
            />
          ) : (
            initials(displayName)
          )}
          <span 
            className="absolute -bottom-1 -right-1 flex h-4 w-4 items-center justify-center rounded-full border bg-white text-[9px] uppercase text-sky-700 font-bold shadow-sm"
            style={{ borderColor: "var(--app-border)" }}
          >
            {labels.channelCode.slice(0, 1)}
          </span>
        </div>
        <div className="min-w-0 flex-1 pr-6">
          <div className="flex items-start justify-between gap-2">
            <div className={clsx("truncate text-sm", unread ? "font-bold" : "font-semibold")}>{displayName}</div>
            <div className="shrink-0 text-[11px]" style={{ color: "var(--app-muted)" }}>
              {relativeTime(conversation.last_message_at)}
            </div>
          </div>
          <div className="mt-1 truncate text-xs" style={{ color: "var(--app-muted)" }}>
            {labels.channelName} · {labels.accountName}
          </div>
          <div className="mt-1 flex items-center justify-between gap-2">
            <div className={clsx("truncate text-xs flex-1", unread && "font-semibold")} style={{ color: unread ? "var(--app-text)" : "var(--app-muted)" }}>
              {conversation.last_message_text || "Chưa có preview tin nhắn"}
            </div>
            {unread && conversation.unread_count > 0 ? (
              <span className="shrink-0 flex h-5 min-w-5 items-center justify-center rounded-full bg-red-600 px-1 text-[10px] font-bold text-white animate-pulse">
                {conversation.unread_count}
              </span>
            ) : null}
          </div>
          <div className="mt-2 flex items-center gap-1">
            {(conversation.tags || []).slice(0, 2).map((tag) => (
              <span key={tag} className="inline-flex items-center rounded-full bg-[var(--app-surface-muted)] px-2 py-0.5 text-[11px] text-[var(--app-muted-strong)] border" style={{ borderColor: "var(--app-border)" }}>
                {tag === "trash" && <Trash2 className="h-3 w-3 text-red-500 mr-0.5 shrink-0" />}
                {tag}
              </span>
            ))}
          </div>
        </div>
      </button>

      {/* Nút ... More Actions */}
      <div className="absolute right-2 top-3 z-10 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity duration-200">
        <button
          ref={menuButtonRef}
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            updateDropdownPosition();
            setShowDropdown((open) => !open);
          }}
          className="flex h-8 w-8 items-center justify-center rounded-full border bg-[var(--app-surface)] shadow-sm hover:bg-[var(--app-surface-hover)] hover:scale-105 active:scale-95 transition-all"
          style={{ borderColor: "var(--app-border)" }}
        >
          <MoreHorizontal className="h-4 w-4" style={{ color: "var(--app-text)" }} />
        </button>
      </div>

      {/* Dropdown Menu */}
      {showDropdown && createPortal(
        <div
          ref={dropdownRef}
          className="fixed w-52 rounded-lg border bg-[var(--app-surface)] py-1.5 shadow-lg animate-in fade-in slide-in-from-top-1 duration-150"
          style={{ borderColor: "var(--app-border)", zIndex: 110, top: dropdownPosition.top, left: dropdownPosition.left }}
          onClick={(e) => e.stopPropagation()}
        >
          <button
            type="button"
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs font-medium hover:bg-[var(--app-surface-hover)] transition"
            onClick={() => setShowDropdown(false)}
          >
            <Pin className="h-3.5 w-3.5" style={{ color: "var(--app-muted)" }} />
            <span>Ghim hội thoại</span>
          </button>
          <button
            type="button"
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs font-medium hover:bg-[var(--app-surface-hover)] transition"
            onClick={() => setShowDropdown(false)}
          >
            <Tag className="h-3.5 w-3.5" style={{ color: "var(--app-muted)" }} />
            <span>Phân loại</span>
          </button>
          <button
            type="button"
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs font-medium hover:bg-[var(--app-surface-hover)] transition"
            onClick={() => setShowDropdown(false)}
          >
            <Mail className="h-3.5 w-3.5" style={{ color: "var(--app-muted)" }} />
            <span>Đánh dấu chưa đọc</span>
          </button>
          <button
            type="button"
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs font-medium hover:bg-[var(--app-surface-hover)] transition"
            onClick={() => setShowDropdown(false)}
          >
            <Users className="h-3.5 w-3.5" style={{ color: "var(--app-muted)" }} />
            <span>Thêm vào nhóm</span>
          </button>
          <button
            type="button"
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs font-medium hover:bg-[var(--app-surface-hover)] transition"
            onClick={() => setShowDropdown(false)}
          >
            <VolumeX className="h-3.5 w-3.5" style={{ color: "var(--app-muted)" }} />
            <span>Tắt thông báo</span>
          </button>
          <button
            type="button"
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs font-medium hover:bg-[var(--app-surface-hover)] transition"
            onClick={() => setShowDropdown(false)}
          >
            <EyeOff className="h-3.5 w-3.5" style={{ color: "var(--app-muted)" }} />
            <span>Ẩn trò chuyện</span>
          </button>
          <button
            type="button"
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs font-medium hover:bg-[var(--app-surface-hover)] transition"
            onClick={() => setShowDropdown(false)}
          >
            <Timer className="h-3.5 w-3.5" style={{ color: "var(--app-muted)" }} />
            <span>Tin nhắn tự xoá</span>
          </button>
          
          <div className="my-1 border-t" style={{ borderColor: "var(--app-border)" }} />
          
          <button
            type="button"
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs font-semibold text-red-600 hover:bg-red-50 transition"
            onClick={() => {
              setShowDropdown(false);
              onDelete();
            }}
          >
            <Trash2 className="h-3.5 w-3.5 text-red-500" />
            <span>Xoá hội thoại</span>
          </button>
          <button
            type="button"
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs font-medium hover:bg-[var(--app-surface-hover)] transition"
            onClick={() => setShowDropdown(false)}
          >
            <AlertTriangle className="h-3.5 w-3.5" style={{ color: "var(--app-muted)" }} />
            <span>Báo xấu</span>
          </button>
        </div>
      , document.body)}
    </div>
  );
}

function MessageBubble({ message, senderName }: { message: Message; senderName: string }) {
  const outbound = message.direction === "outbound";
  const StatusIcon = message.read_at ? CheckCheck : message.status === "pending" ? Clock3 : Check;
  return (
    <div className={clsx("flex", outbound ? "justify-end" : "justify-start")}>
      <div
        className={clsx("max-w-[78%] rounded-md px-3 py-2 text-sm shadow-sm", outbound ? "bg-sky-600 text-white" : "border bg-[var(--app-surface)]")}
        style={{ borderColor: outbound ? undefined : "var(--app-border)" }}
      >
        {!outbound ? <div className="mb-1 text-xs font-semibold text-[var(--app-muted-strong)]">{senderName}</div> : null}
        <div className="whitespace-pre-wrap break-words">{message.text || "(no text)"}</div>
        <div className={clsx("mt-1 flex flex-wrap items-center justify-end gap-1 text-[11px]", outbound ? "text-sky-100" : "text-[var(--app-muted)]")}>
          <span>{messageStatusLabel(message)}</span>
          <span>·</span>
          <span>{formatMessageDate(message.event_time)}</span>
          {outbound ? (
            <>
              <span>·</span>
              <StatusIcon className="h-3 w-3" />
              {message.read_at ? <span>seen</span> : null}
            </>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function TypingIndicator({ senderName }: { senderName: string }) {
  return (
    <div className="flex justify-start">
      <div className="rounded-md border bg-[var(--app-surface)] px-3 py-2 text-sm shadow-sm" style={{ borderColor: "var(--app-border)" }}>
        <div className="mb-1 text-xs font-semibold text-[var(--app-muted-strong)]">{senderName}</div>
        <div className="flex items-center gap-2 text-[var(--app-muted)]">
          <span>đang nhập</span>
          <span className="flex gap-1">
            <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-current [animation-delay:-0.2s]" />
            <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-current [animation-delay:-0.1s]" />
            <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-current" />
          </span>
        </div>
      </div>
    </div>
  );
}

function InfoTab({ conversation, labels }: { conversation: Conversation; labels: { accountName: string; channelName: string } }) {
  return (
    <div className="space-y-3 text-sm">
      <InfoRow label="Customer" value={displayCustomer(conversation)} />
      <InfoRow label="Contact ID" value={conversation.customer_ref || conversation.external_conversation_id} />
      <InfoRow label="Channel" value={labels.channelName} />
      <InfoRow label="Account" value={labels.accountName} />
      <InfoRow label="Status" value={conversation.status} />
      <InfoRow label="Assigned user" value={conversation.assigned_user_id || "Unassigned"} />
      <InfoRow label="Assigned team" value={conversation.assigned_team_id || "Unassigned"} />
      <InfoRow label="External ID" value={conversation.external_conversation_id} />
      <InfoRow label="Created" value={formatDate(conversation.created_at)} />
      <InfoRow label="Last message" value={formatDate(conversation.last_message_at)} />
    </div>
  );
}

function TagsTab({
  conversation,
  draft,
  saving,
  setDraft,
  onAdd,
  onRemove,
}: {
  conversation: Conversation;
  draft: string;
  saving: boolean;
  setDraft: (value: string) => void;
  onAdd: (event: FormEvent) => void;
  onRemove: (tag: string) => void;
}) {
  return (
    <div>
      <form className="mb-3 flex gap-2" onSubmit={onAdd}>
        <div className="relative min-w-0 flex-1">
          <Hash className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2" style={{ color: "var(--app-muted)" }} />
          <input className="field h-9 pl-9" value={draft} onChange={(event) => setDraft(event.target.value)} placeholder="Add tag" />
        </div>
        <button className="btn h-9 w-9 px-0" disabled={saving || !draft.trim()} title="Add tag">
          {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
        </button>
      </form>
      <div className="flex flex-wrap gap-2">
        {(conversation.tags || []).length === 0 ? (
          <div className="text-sm" style={{ color: "var(--app-muted)" }}>
            No tags yet.
          </div>
        ) : (
          conversation.tags.map((tag) => (
            <span key={tag} className="inline-flex items-center gap-1 rounded-full bg-[var(--app-accent-soft-bg)] px-2 py-1 text-xs font-medium text-[var(--app-accent-soft-fg)]">
              {tag === "trash" && <Trash2 className="h-3 w-3 text-red-500 mr-0.5 shrink-0" />}
              {tag}
              <button className="rounded-full p-0.5 hover:bg-black/10" onClick={() => onRemove(tag)} title={`Remove ${tag}`}>
                <X size={12} />
              </button>
            </span>
          ))
        )}
      </div>
    </div>
  );
}

function HistoryTab({ conversation, messages }: { conversation: Conversation; messages: Message[] }) {
  const inbound = messages.filter((message) => message.direction === "inbound").length;
  const outbound = messages.filter((message) => message.direction === "outbound").length;
  return (
    <div className="space-y-3 text-sm">
      <InfoRow label="Messages loaded" value={String(messages.length)} />
      <InfoRow label="Inbound" value={String(inbound)} />
      <InfoRow label="Outbound" value={String(outbound)} />
      <InfoRow label="Unread" value={String(conversation.unread_count || 0)} />
      <InfoRow label="Updated" value={formatDate(conversation.updated_at)} />
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border p-3" style={{ borderColor: "var(--app-border)" }}>
      <div className="text-xs" style={{ color: "var(--app-muted)" }}>
        {label}
      </div>
      <div className="mt-1 break-words font-medium">{value}</div>
    </div>
  );
}

function displayCustomer(conversation: Conversation) {
  return conversation.customer_name || cleanCustomerRef(conversation.customer_ref) || cleanCustomerRef(conversation.external_conversation_id) || "Unknown customer";
}

function cleanCustomerRef(value?: string) {
  if (!value) return "";
  const trimmed = value.trim();
  const withoutDevice = trimmed.split(":")[0] || trimmed;
  const withoutDomain = withoutDevice.split("@")[0] || withoutDevice;
  return withoutDomain || trimmed;
}

function initials(value: string) {
  return value
    .split(/[^\p{L}\p{N}]+/u)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join("") || "C";
}

function formatDate(value: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat("vi-VN", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

function formatMessageDate(value: string) {
  if (!value) return "";
  return new Intl.DateTimeFormat("vi-VN", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function messageStatusLabel(message: Message) {
  if (message.read_at) return "seen";
  if (message.delivered_at) return "delivered";
  if (message.sent_at || message.status === "sent") return "sent";
  if (message.status === "pending") return "sending";
  return message.status || "sent";
}

function relativeTime(value: string) {
  if (!value) return "";
  const diffMs = Date.now() - new Date(value).getTime();
  const minutes = Math.max(0, Math.round(diffMs / 60000));
  if (minutes < 1) return "now";
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours}h`;
  return `${Math.round(hours / 24)}d`;
}
