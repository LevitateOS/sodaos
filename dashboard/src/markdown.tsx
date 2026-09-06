import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Link } from "react-router-dom";

interface RepositoryContext { route: string; ref: string; path: string }
// Only normal HTTP(S) navigation and repository-relative links. No image fetch,
// raw HTML, plugin execution, iframe, credential forwarding or active download.
export function Markdown({ text, repository }: { text: string; repository?: RepositoryContext }) {
  return <div className="soda-markdown"><ReactMarkdown skipHtml remarkPlugins={[remarkGfm]} components={{
    img: ({ alt }) => <span>[Image not loaded{alt ? `: ${alt}` : ""}]</span>,
    a: ({ href, children }) => {
      if (!href) return <span>{children}</span>;
      if (href.startsWith("#")) return <a href={href}>{children}</a>;
      try {
        const base = `https://repository.invalid/${(repository?.path ?? "").split("/").map(encodeURIComponent).join("/")}`;
        const target = new URL(href, base);
        if (target.protocol !== "https:" && target.protocol !== "http:") return <span>{children}</span>;
        if (target.username || target.password) return <span>{children}</span>;
        if (target.origin === "https://repository.invalid") {
          if (!repository) return <span>{children}</span>;
          const path = decodeURIComponent(target.pathname.replace(/^\//, ""));
          const query = new URLSearchParams({ ref: repository.ref, path });
          return <Link to={`${repository.route}?${query}${target.hash}`}>{children}</Link>;
        }
        return <a href={target.href} rel="noreferrer noopener">{children}</a>;
      } catch { return <span>{children}</span>; }
    },
  }}>{text}</ReactMarkdown></div>;
}
