"use client";

import { useRouter } from "next/navigation";
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import type { Profile } from "@/lib/types";

const TOKEN_KEY = "omni_cms_access_token";

type AuthContextValue = {
  token: string | null;
  profile: Profile | null;
  loading: boolean;
  error: string;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  refreshProfile: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [token, setToken] = useState<string | null>(null);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const loadProfile = useCallback(async (nextToken: string) => {
    const nextProfile = await api.profile(nextToken);
    setProfile(nextProfile);
  }, []);

  useEffect(() => {
    const stored = window.localStorage.getItem(TOKEN_KEY);
    if (!stored) {
      setLoading(false);
      return;
    }
    setToken(stored);
    loadProfile(stored)
      .catch(() => {
        window.localStorage.removeItem(TOKEN_KEY);
        setToken(null);
        setProfile(null);
      })
      .finally(() => setLoading(false));
  }, [loadProfile]);

  const login = useCallback(
    async (email: string, password: string) => {
      setError("");
      const result = await api.login(email, password);
      window.localStorage.setItem(TOKEN_KEY, result.access_token);
      setToken(result.access_token);
      await loadProfile(result.access_token);
      router.push("/dashboard");
    },
    [loadProfile, router],
  );

  const logout = useCallback(() => {
    window.localStorage.removeItem(TOKEN_KEY);
    setToken(null);
    setProfile(null);
    router.push("/login");
  }, [router]);

  const refreshProfile = useCallback(async () => {
    if (!token) return;
    await loadProfile(token);
  }, [loadProfile, token]);

  const value = useMemo(
    () => ({ token, profile, loading, error, login, logout, refreshProfile }),
    [token, profile, loading, error, login, logout, refreshProfile],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return context;
}
