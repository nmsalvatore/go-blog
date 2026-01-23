document.querySelectorAll("textarea").forEach((textarea) => {
    textarea.addEventListener("input", () => {
        autoResize(textarea);
    });

    autoResize(textarea);
});

function autoResize(textarea) {
    const scrollTop = window.scrollY;
    const scrollLeft = window.scrollX;

    textarea.style.height = "1px";
    const computed = getComputedStyle(textarea);
    const border = parseFloat(computed.borderTopWidth) + parseFloat(computed.borderBottomWidth);
    const newHeight = textarea.scrollHeight + border;
    textarea.style.height = newHeight + "px";

    window.scrollTo(scrollLeft, scrollTop);
}
