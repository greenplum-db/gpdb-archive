//---------------------------------------------------------------------------
//	Greenplum Database
//	Copyright (c) 2023 VMware, Inc. or its affiliates. All Rights Reserved.
//
//	@filename:
//		CPlanHint.cpp
//
//	@doc:
//		Container of plan hint objects
//---------------------------------------------------------------------------

#include "gpopt/hints/CPlanHint.h"

#include "gpos/base.h"

using namespace gpopt;

FORCE_GENERATE_DBGSTR(CPlanHint);

//---------------------------------------------------------------------------
//	@function:
//		CPlanHint::CPlanHint
//---------------------------------------------------------------------------
CPlanHint::CPlanHint(CMemoryPool *mp)
	: m_mp(mp),
	  m_scan_hints(GPOS_NEW(mp) ScanHintList(mp)),
	  m_row_hints(GPOS_NEW(mp) RowHintList(mp))
{
}

CPlanHint::~CPlanHint()
{
	m_scan_hints->Release();
	m_row_hints->Release();
}

void
CPlanHint::AddHint(CScanHint *hint)
{
	m_scan_hints->Append(hint);
}

void
CPlanHint::AddHint(CRowHint *hint)
{
	m_row_hints->Append(hint);
}


CScanHint *
CPlanHint::GetScanHint(const char *relname)
{
	CWStringConst *name = GPOS_NEW(m_mp) CWStringConst(m_mp, relname);
	CScanHint *hint = GetScanHint(name);
	GPOS_DELETE(name);
	return hint;
}

CScanHint *
CPlanHint::GetScanHint(const CWStringBase *name)
{
	for (ULONG ul = 0; ul < m_scan_hints->Size(); ul++)
	{
		CScanHint *hint = (*m_scan_hints)[ul];
		if (name->Equals(hint->GetName()))
		{
			return hint;
		}
	}
	return nullptr;
}

//---------------------------------------------------------------------------
//	@function:
//		CPlanHint::GetRowHint
//
//	@doc:
//		Given a set of table descriptors, find a matching CRowHint.  A match
//		means that the table alias names used to describe the CRowHint equals
//		the set of table alias names provided by the given set of table
//		descriptors.
//
//---------------------------------------------------------------------------
CRowHint *
CPlanHint::GetRowHint(CTableDescriptorHashSet *ptabdescs)
{
	GPOS_ASSERT(ptabdescs->Size() > 0);

	StringPtrArray *aliases = GPOS_NEW(m_mp) StringPtrArray(m_mp);
	CTableDescriptorHashSetIter hsiter(ptabdescs);

	while (hsiter.Advance())
	{
		const CTableDescriptor *ptabdesc = hsiter.Get();
		aliases->Append(GPOS_NEW(m_mp) CWStringConst(
			m_mp, ptabdesc->Name().Pstr()->GetBuffer()));
	}
	// Row hint aliases are sorted because the hint is order agnostic.
	aliases->Sort(CWStringBase::Compare);

	CRowHint *matching_hint = nullptr;
	for (ULONG ul = 0; ul < m_row_hints->Size(); ul++)
	{
		CRowHint *hint = (*m_row_hints)[ul];
		if (aliases->Equals(hint->GetAliasNames()))
		{
			matching_hint = hint;
			break;
		}
	}
	aliases->Release();
	return matching_hint;
}

IOstream &
CPlanHint::OsPrint(IOstream &os) const
{
	os << "PlanHint: [";
	if (nullptr == m_scan_hints)
	{
		os << "]";
		return os;
	}

	os << "\n";
	for (ULONG ul = 0; ul < m_scan_hints->Size(); ul++)
	{
		os << "  ";
		(*m_scan_hints)[ul]->OsPrint(os) << "\n";
	}

	for (ULONG ul = 0; ul < m_row_hints->Size(); ul++)
	{
		os << "  ";
		(*m_row_hints)[ul]->OsPrint(os) << "\n";
	}
	os << "]";
	return os;
}

void
CPlanHint::Serialize(CXMLSerializer *xml_serializer) const
{
	for (ULONG ul = 0; ul < m_scan_hints->Size(); ul++)
	{
		(*m_scan_hints)[ul]->Serialize(xml_serializer);
	}

	for (ULONG ul = 0; ul < m_row_hints->Size(); ul++)
	{
		(*m_row_hints)[ul]->Serialize(xml_serializer);
	}
}


// EOF
