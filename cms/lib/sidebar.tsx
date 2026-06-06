"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";

const SIDEBAR_KEY = "omni_cms_sidebar_open";

type SidebarContextValue = {
  isOpen: boolean;
  toggle: () => void;
  setIsOpen: (open: boolean) => void;
};

const SidebarContext = createContext<SidebarContextValue | undefined>(undefined);

export function SidebarProvider({ children }: { children: React.ReactNode }) {
  const [isOpen, setIsOpenState] = useState(true);

  useEffect(() => {
    const stored = window.localStorage.getItem(SIDEBAR_KEY);
    if (stored !== null) {
      setIsOpenState(stored === "true");
    }
  }, []);

  function setIsOpen(open: boolean) {
    window.localStorage.setItem(SIDEBAR_KEY, String(open));
    setIsOpenState(open);
  }

  function toggle() {
    setIsOpen(!isOpen);
  }

  const value = useMemo(() => ({ isOpen, toggle, setIsOpen }), [isOpen]);
  return <SidebarContext.Provider value={value}>{children}</SidebarContext.Provider>;
}

export function useSidebar() {
  const context = useContext(SidebarContext);
  if (!context) {
    throw new Error("useSidebar must be used inside SidebarProvider");
  }
  return context;
}
