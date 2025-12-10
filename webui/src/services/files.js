export function fileToBase64(file) {
	return new Promise((resolve, reject) => {
		if (!file) {
			resolve("");
			return;
		}
		const reader = new FileReader();
		reader.onload = () => {
			const result = reader.result || "";
			const base64 = typeof result === "string" && result.includes(",")
				? result.split(",")[1]
				: result;
			resolve(base64 || "");
		};
		reader.onerror = reject;
		reader.readAsDataURL(file);
	});
}

export function cropImageSquare(dataUrl, {size = 320, offsetX = 0, offsetY = 0, scale = 1} = {}) {
	return new Promise((resolve, reject) => {
		if (!dataUrl) {
			resolve("");
			return;
		}
		const img = new Image();
		img.onload = () => {
			const canvas = document.createElement("canvas");
			canvas.width = size;
			canvas.height = size;
			const ctx = canvas.getContext("2d");
			const baseSide = Math.min(img.width, img.height);
			const srcSide = baseSide / scale;
			const maxOffset = baseSide - srcSide;
			const sx = (img.width - srcSide) / 2 + offsetX * maxOffset;
			const sy = (img.height - srcSide) / 2 + offsetY * maxOffset;
			ctx.drawImage(img, sx, sy, srcSide, srcSide, 0, 0, size, size);
			const result = canvas.toDataURL("image/png");
			const base64 = result.includes(",") ? result.split(",")[1] : result;
			resolve(base64 || "");
		};
		img.onerror = reject;
		img.src = dataUrl;
	});
}

// Ensures a base64 image string can be used directly in <img> or CSS by adding the data URL prefix.
export function base64ImageToDataUrl(base64) {
	if (!base64 || typeof base64 !== "string") {
		return "";
	}
	const trimmed = base64.trim();
	if (trimmed.startsWith("data:image")) {
		return trimmed;
	}
	const sig = trimmed.slice(0, 10);
	let mime = "image/png";
	if (sig.startsWith("iVBORw0KGgo")) {
		mime = "image/png";
	} else if (sig.startsWith("/9j/")) {
		mime = "image/jpeg";
	} else if (sig.startsWith("R0lGOD")) {
		mime = "image/gif";
	}
	return `data:${mime};base64,${trimmed}`;
}
