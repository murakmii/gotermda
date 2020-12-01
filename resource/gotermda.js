window.onload = function() {
    var terminal = document.getElementById("terminal");

    fetch("/open", {method: "POST"})
        .then(response => response.json())
        .then(data => {
            var stream = new EventSource("/read/" + data.terminal_id);
            stream.onmessage = function (e) {
                terminal.innerText = atob(e.data)
            };
        })
};
