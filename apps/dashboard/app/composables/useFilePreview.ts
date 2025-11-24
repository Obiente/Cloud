/**
 * Composable for detecting file preview types
 */

export type FilePreviewType =
  | "text"
  | "image"
  | "video"
  | "audio"
  | "pdf"
  | "zip"
  | "binary";

/**
 * Detect the preview type for a file based on path, MIME type, and size
 */
export function detectFilePreviewType(
  path: string,
  mimeType?: string,
  fileSize: number = 0
): FilePreviewType {
  // Empty files (0 bytes) should default to text unless they have a known binary extension
  if (fileSize === 0) {
    // Check if it has a known binary extension
    const ext = path.split(".").pop()?.toLowerCase() || "";
    const binaryExts = [
      "exe",
      "dll",
      "so",
      "dylib",
      "bin",
      "app",
      "deb",
      "rpm",
      "pkg",
      "dmg",
      "iso",
      "img",
    ];
    if (binaryExts.includes(ext)) {
      return "binary";
    }
    // Default empty files to text so they can be edited
    return "text";
  }

  // Extract filename and extension early for use in multiple checks
  const filename = path.split("/").pop()?.toLowerCase() || "";
  const ext = path.split(".").pop()?.toLowerCase() || "";
  const lowerPath = path.toLowerCase();

  // Common text filenames (game server configs and other files without extensions)
  // Check these BEFORE MIME type to override incorrect MIME type detection
  const commonTextFilenames = [
    "server.properties",
    "server.properties.tmp",
    "banned-players.json",
    "banned-ips.json",
    "ops.json",
    "whitelist.json",
    "usercache.json",
    "eula.txt",
    "bukkit.yml",
    "spigot.yml",
    "paper.yml",
    "server.config",
    "serverconfig.xml",
    "valheim_server.config",
    "server.cfg",
    "serverconfig.txt",
    "run.bat",
    "run.sh",
    "start.sh",
    "start.bat",
    "launch.sh",
    "launch.bat",
  ];

  // Common text file paths (especially system files without extensions)
  const commonTextPaths = [
    "/etc/",
    "/etc/profile",
    "/etc/passwd",
    "/etc/group",
    "/etc/hosts",
    "/etc/fstab",
    "/etc/resolv.conf",
    "/etc/ssh/",
    "/etc/nginx/",
    "/etc/apache/",
    "/var/log/",
    "/home/",
    "/root/",
    "/opt/",
    "/usr/local/",
    ".env",
    ".gitignore",
    ".dockerignore",
    "Dockerfile",
    "docker-compose",
    "Makefile",
    "README",
    "CHANGELOG",
    "LICENSE",
  ];

  // Check if filename matches common text filenames (before MIME type check)
  if (commonTextFilenames.some((name) => filename === name.toLowerCase())) {
    return "text";
  }

  // Check if path matches common text file patterns (before MIME type check)
  if (
    commonTextPaths.some((pattern) => lowerPath.includes(pattern.toLowerCase()))
  ) {
    return "text";
  }

  // Check MIME type (but don't trust it blindly for known text files)
  if (mimeType) {
    if (mimeType.startsWith("image/")) return "image";
    if (mimeType.startsWith("video/")) return "video";
    if (mimeType.startsWith("audio/")) return "audio";
    if (mimeType === "application/pdf") return "pdf";
    if (
      mimeType === "application/zip" ||
      mimeType === "application/x-zip-compressed" ||
      mimeType === "application/x-java-archive" ||
      mimeType === "application/x-war" ||
      mimeType === "application/x-ear"
    )
      return "zip";
    // Text MIME types
    if (
      mimeType.startsWith("text/") ||
      mimeType.includes("json") ||
      mimeType.includes("xml") ||
      mimeType.includes("javascript") ||
      mimeType.includes("css") ||
      mimeType.includes("html")
    ) {
      return "text";
    }
    // If it's a binary MIME type, return binary (but only if we haven't already identified it as text)
    if (
      mimeType.startsWith("application/") &&
      !mimeType.includes("json") &&
      !mimeType.includes("xml") &&
      mimeType !== "application/pdf"
    ) {
      return "binary";
    }
  }

  // Fallback to file extension
  const imageExts = [
    "jpg",
    "jpeg",
    "png",
    "gif",
    "webp",
    "svg",
    "bmp",
    "ico",
    "tiff",
    "tif",
  ];
  const videoExts = ["mp4", "webm", "ogg", "mov", "avi", "mkv", "flv", "wmv"];
  const audioExts = [
    "mp3",
    "wav",
    "ogg",
    "aac",
    "flac",
    "m4a",
    "wma",
    "opus",
  ];

  const textExts = [
    "txt",
    "md",
    "json",
    "yaml",
    "yml",
    "xml",
    "html",
    "htm",
    "css",
    "js",
    "jsx",
    "ts",
    "tsx",
    "py",
    "go",
    "rs",
    "java",
    "c",
    "cpp",
    "h",
    "hpp",
    "sh",
    "bash",
    "zsh",
    "fish",
    "sql",
    "log",
    "conf",
    "config",
    "ini",
    "env",
    "dockerfile",
    "makefile",
    "gitignore",
    "gitattributes",
    "editorconfig",
    "prettierrc",
    "eslintrc",
  ];

  if (imageExts.includes(ext)) return "image";
  if (videoExts.includes(ext)) return "video";
  if (audioExts.includes(ext)) return "audio";
  if (ext === "pdf") return "pdf";
  if (ext === "zip" || ext === "jar" || ext === "war" || ext === "ear")
    return "zip";
  if (textExts.includes(ext)) return "text";

  // Default to binary for unknown types
  return "binary";
}

