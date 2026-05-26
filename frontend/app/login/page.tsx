"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useMemo, useState } from "react";

import { dummyLogin, login } from "@/src/api/auth";
import { toUserErrorMessage } from "@/src/components/formatters";
import { useAuth } from "@/src/auth/AuthProvider";
import type { UserRole } from "@/src/types/api";

export default function LoginPage() {
  const { ready, session, applyToken } = useAuth();
  const router = useRouter();
  const nextPath = useMemo(() => {
    if (typeof window === "undefined") {
      return "/rooms";
    }
    const query = new URLSearchParams(window.location.search);
    return query.get("next") ?? "/rooms";
  }, []);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (ready && session) {
      router.replace(nextPath);
    }
  }, [nextPath, ready, router, session]);

  const handleLogin = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const response = await login(email, password);
      if (!applyToken(response.token)) {
        setError("Получен некорректный токен.");
        return;
      }
      router.push(nextPath);
    } catch (err) {
      setError(toUserErrorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const handleDummyLogin = async (role: UserRole) => {
    setBusy(true);
    setError(null);
    try {
      const response = await dummyLogin(role);
      if (!applyToken(response.token)) {
        setError("Получен некорректный токен.");
        return;
      }
      router.push(nextPath);
    } catch (err) {
      setError(toUserErrorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-6">
      <h2 className="text-xl font-semibold">Login</h2>
      <p className="mt-1 text-sm text-zinc-400">Вход в backend для ручного теста.</p>

      <form className="mt-4 space-y-3" onSubmit={handleLogin}>
        <label className="block text-sm">
          <span className="mb-1 block text-zinc-300">Email</span>
          <input
            type="email"
            className="w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-zinc-100"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            required
          />
        </label>
        <label className="block text-sm">
          <span className="mb-1 block text-zinc-300">Password</span>
          <input
            type="password"
            className="w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-zinc-100"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
          />
        </label>
        <button
          type="submit"
          className="rounded-md bg-emerald-600 px-4 py-2 text-sm text-white hover:bg-emerald-500 disabled:opacity-60"
          disabled={busy}
        >
          Login
        </button>
      </form>

      <div className="mt-5 flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded-md bg-zinc-700 px-3 py-2 text-sm hover:bg-zinc-600 disabled:opacity-60"
          onClick={() => handleDummyLogin("admin")}
          disabled={busy}
        >
          Login as admin (dummyLogin)
        </button>
        <button
          type="button"
          className="rounded-md bg-zinc-700 px-3 py-2 text-sm hover:bg-zinc-600 disabled:opacity-60"
          onClick={() => handleDummyLogin("user")}
          disabled={busy}
        >
          Login as user (dummyLogin)
        </button>
      </div>

      {error ? <p className="mt-4 text-sm text-red-300">{error}</p> : null}

      <p className="mt-4 text-sm text-zinc-300">
        Нет аккаунта?{" "}
        <Link href="/register" className="text-emerald-400 hover:text-emerald-300">
          Register
        </Link>
      </p>
    </div>
  );
}
