// This is a polyfill for FireFox and Safari
if (!WebAssembly.instantiateStreaming) {
    WebAssembly.instantiateStreaming = async (resp, importObject) => {
        const source = await (await resp).arrayBuffer()
        return await WebAssembly.instantiate(source, importObject)
    }
}

// Promise to load the wasm file
function loadWasm(path) {
    const go = new Go()

    return new Promise((resolve, reject) => {
        WebAssembly.instantiateStreaming(fetch(path), go.importObject)
            .then(result => {
                go.run(result.instance)
                resolve(result.instance)
            })
            .catch(error => {
                reject(error)
            })
    })
}

// Load the wasm file
loadWasm("assets/wasm/k3d_smc_lib.wasm").then(wasm => {
    console.log("k3d_smc_lib.wasm is loaded 👋")
    document.getElementById("generateButton").style.display = "inline"
    document.getElementById("generateButtonLoading").style.display = "none"
}).catch(error => {
    console.log("ouch", error)
})
