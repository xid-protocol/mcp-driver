import Chat from "../components/Chat";
import "./globals.css";

export default function Page() {
	return (
		<main className="max-w-3xl mx-auto p-4">
			<h1 className="text-2xl font-semibold mb-4">SSE Chat</h1>
			<Chat />
		</main>
	);
}



