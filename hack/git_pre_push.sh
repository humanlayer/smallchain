CHANGED_FILES=$(git diff --name-only upstream/main)

if echo "${CHANGED_FILES}" | grep -qE '^kubechain/'; then
	echo ": -- 🚀 kubechain --"
	make -C kubechain test lint
else
	echo ": -- ⏭️ kubechain --"
fi

