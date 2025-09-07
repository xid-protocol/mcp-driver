import "./globals.css";

export const metadata = {
	title: "Multi Task Chat",
	description: "SSE Chat UI",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
	return (
		<html lang="zh-CN">
			<body className="min-h-screen bg-gray-50 text-gray-900">
				{children}
			</body>
		</html>
	);
}
