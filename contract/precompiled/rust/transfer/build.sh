cargo +nightly build --target wasm32-unknown-unknown --release
cp target/wasm32-unknown-unknown/release/*.wasm ../../wasm/transfer_contract.wasm
pushd ../../wasm/
wapm run wasm2wat transfer_contract.wasm > transfer_contract.wast
popd
