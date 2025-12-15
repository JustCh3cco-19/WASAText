const BASE_TITLE = "WASAText";

export function setPageTitle(title) {
	const safeTitle = (title || "").toString().trim();
	document.title = safeTitle ? `${safeTitle} · ${BASE_TITLE}` : BASE_TITLE;
}

export {BASE_TITLE};
