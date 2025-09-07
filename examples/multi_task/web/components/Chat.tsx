"use client";

import { useEffect, useMemo, useRef, useState } from "react";

function getEnv(key: string): string | undefined {
	if (typeof window === "undefined") return undefined;
	// @ts-ignore
	return process.env[key];
}

// 可在 .env.local 配置:
// - NEXT_PUBLIC_API_BASE: 后端基地址，默认同源
// - NEXT_PUBLIC_CHAT_PATH: 聊天流式接口路径，默认 /api/chat/stream
export default function Chat() {
	const [messages, setMessages] = useState<string[]>([]);
	const [input, setInput] = useState("");
	const [loading, setLoading] = useState(false);
	const abortRef = useRef<AbortController | null>(null);

	const apiUrl = useMemo(() => {
		const base = getEnv("NEXT_PUBLIC_API_BASE") || "";
		const path = getEnv("NEXT_PUBLIC_CHAT_PATH") || "/api/chat/stream";
		return `${base}${path}`;
	}, []);

	useEffect(() => {
		return () => {
			abortRef.current?.abort();
		};
	}, []);

	async function handleSend() {
		if (!input.trim() || loading) return;
		setMessages((prev) => [...prev, `你: ${input}`]);
		setLoading(true);

		const controller = new AbortController();
		abortRef.current = controller;

		try {
			const resp = await fetch(apiUrl, {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
				},
				body: JSON.stringify({ message: input }),
				signal: controller.signal,
			});

			if (!resp.ok) {
				throw new Error(`请求失败: ${resp.status}`);
			}

			const reader = resp.body?.getReader();
			const decoder = new TextDecoder("utf-8");
			let assistant = "";

			if (!reader) throw new Error("无法读取响应流");

			while (true) {
				const { done, value } = await reader.read();
				if (done) break;
				const chunk = decoder.decode(value, { stream: true });
				for (const line of chunk.split(/\r?\n/)) {
					if (line.startsWith("data:")) {
						const data = line.slice(5).trim();
						if (data === "[DONE]") {
							continue;
						}
						if (data) {
							assistant += data;
							setMessages((prev) => {
								const next = [...prev];
								if (next.length && next[next.length - 1].startsWith("助理:")) {
									next[next.length - 1] = `助理: ${assistant}`;
								} else {
									next.push(`助理: ${assistant}`);
								}
								return next;
							});
						}
					}
				}
			}
		} catch (err: any) {
			setMessages((prev) => [...prev, `错误: ${err.message || String(err)}`]);
		} finally {
			setLoading(false);
			setInput("");
			abortRef.current = null;
		}
	}

	return (
		<div className="space-y-4">
			<div className="border rounded p-3 h-80 overflow-auto bg-white">
				{messages.length === 0 ? (
					<p className="text-gray-400">开始对话吧~</p>
				) : (
					<ul className="space-y-2">
						{messages.map((m, i) => (
							<li key={i} className="whitespace-pre-wrap text-sm">
								{m}
							</li>
						))}
					</ul>
				)}
			</div>
			<div className="flex gap-2">
				<input
					type="text"
					value={input}
					onChange={(e) => setInput(e.target.value)}
					placeholder="输入消息..."
					className="flex-1 border rounded px-3 py-2 text-sm"
					disabled={loading}
				/>
				<button
					onClick={handleSend}
					disabled={loading}
					className="px-4 py-2 text-sm rounded bg-blue-600 text-white disabled:opacity-50"
				>
					{loading ? "发送中..." : "发送"}
				</button>
			</div>
		</div>
	);
}
