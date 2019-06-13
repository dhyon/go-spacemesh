cargo +nightly build --target wasm32-unknown-unknown --release
cp target/wasm32-unknown-unknown/release/*.wasm ../../wasm/nop_contract.wasm
pushd ../../wasm/
wapm run wasm2wat nop_contract.wasm > nop_contract.wast
popd
