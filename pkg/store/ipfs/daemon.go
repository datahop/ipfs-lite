package ipfs

func (I *IPFSNode) Stop() {
	I.peer.Cancel()
	select {
	case _, ok := <-I.peer.Stopped:
		if !ok {
			return
		}
		close(I.peer.Stopped)
	}
}
