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
