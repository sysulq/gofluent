package ik

import (
	"log"
	"sync"
	"unsafe"
)

type SlicerNewKeyEventListener func (last Journal, next Journal) error

type Slicer struct {
	journalGroup JournalGroup
	keyGetter func (record FluentRecord) string
	packer RecordPacker
	logger *log.Logger
	keys map[string]bool
	newKeyEventListeners map[uintptr]SlicerNewKeyEventListener
	mtx sync.Mutex
}

func (slicer *Slicer) notifySlicerNewKeyEventListeners(last Journal, next Journal) {
	// lock for slicer must be acquired by caller
	for _, listener := range slicer.newKeyEventListeners {
		err := listener(last, next)
		if err != nil {
			slicer.logger.Printf("error occurred during notifying flush event: %s", err.Error())
		}
	}
}

func (slicer *Slicer) AddNewKeyEventListener(listener SlicerNewKeyEventListener) {
	slicer.mtx.Lock()
	defer slicer.mtx.Unlock()
	// XXX hack!
	slicer.newKeyEventListeners[uintptr(*(*unsafe.Pointer)(unsafe.Pointer(&listener)))] = listener
}

func (slicer *Slicer) Emit(recordSets []FluentRecordSet) error {
	journals := make(map[string]Journal)
	lastJournal := (Journal)(nil)
	for _, recordSet := range recordSets {
		tag := recordSet.Tag
		for _, record := range recordSet.Records {
			fullRecord := FluentRecord {
				tag,
				record.Timestamp,
				record.Data,
			}
			key := slicer.keyGetter(fullRecord)
			data, err := slicer.packer.Pack(fullRecord)
			if err != nil {
				return err
			}
			journal, ok := journals[key]
			if !ok {
				journal = slicer.journalGroup.GetJournal(key)
				journals[key] = journal
				slicer.mtx.Lock()
				_, ok := slicer.keys[key]
				slicer.keys[key] = true
				slicer.mtx.Unlock()
				if !ok {
					slicer.notifySlicerNewKeyEventListeners(lastJournal, journal)
				}
				lastJournal = journal
			}
			err = journal.Write(data)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func NewSlicer(journalGroup JournalGroup, keyGetter func (record FluentRecord) string, packer RecordPacker, logger *log.Logger) *Slicer {
	keys := make(map[string]bool)
	for _, key := range journalGroup.GetJournalKeys() {
		keys[key] = true
	}
	return &Slicer {
		journalGroup: journalGroup,
		keyGetter: keyGetter,
		packer: packer,
		logger: logger,
		keys: keys,
		newKeyEventListeners: make(map[uintptr]SlicerNewKeyEventListener),
		mtx: sync.Mutex {},
	}
}
