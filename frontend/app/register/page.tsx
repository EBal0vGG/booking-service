"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";

import { login, register } from "@/src/api/auth";
import { useAuth } from "@/src/auth/AuthProvider";
import { toUserErrorMessage } from "@/src/components/formatters";
import type { UserRole } from "@/src/types/api";

export default function RegisterPage() {
  const router = useRouter();
  const { applyToken } = useAuth();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<UserRole>("user");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setMessage(null);
    setError(null);
    try {
      await register(email, password, role);
      const auth = await login(email, password);
      if (applyToken(auth.token)) {
        router.push("/rooms");
        return;
      }
      setMessage("Пользователь создан. Выполните вход на странице login.");
      router.push("/login");
    } catch (err) {
      setError(toUserErrorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-6">
      <h2 className="text-xl font-semibold">Register</h2>
      <p className="mt-1 text-sm text-zinc-400">Создание пользователя для тестов.</p>

      <form className="mt-4 space-y-3" onSubmit={handleSubmit}>
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
        <label className="block text-sm">
          <span className="mb-1 block text-zinc-300">Role</span>
          <select
            className="w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-zinc-100"
            value={role}
            onChange={(event) => setRole(event.target.value as UserRole)}
          >
            <option value="user">user</option>
            <option value="admin">admin</option>
          </select>
        </label>

        <button
          type="submit"
          className="rounded-md bg-emerald-600 px-4 py-2 text-sm text-white hover:bg-emerald-500 disabled:opacity-60"
          disabled={busy}
        >
          Register
        </button>
      </form>

      {message ? <p className="mt-4 text-sm text-emerald-300">{message}</p> : null}
      {error ? <p className="mt-2 text-sm text-red-300">{error}</p> : null}

      <p className="mt-4 text-sm text-zinc-300">
        Уже есть аккаунт?{" "}
        <Link href="/login" className="text-emerald-400 hover:text-emerald-300">
          Login
        </Link>
      </p>
    </div>
  );
}
